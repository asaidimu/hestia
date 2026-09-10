import { describe, expect, it, beforeAll, afterAll } from "vitest"
import { makeClient, collectionSchema } from "../tests/helpers"
import { HestiaDataView } from "./data-view"
import type { HestiaCollection } from "./collection"

/**
 * Short unique ids on purpose: go-anansi derives a view's physical table
 * name from the view name truncated to ~18 sanitized chars, so long names
 * with a common prefix (e.g. `e2e_view_snap-<ts>-<rand>`) collide onto one
 * physical table across runs and CTAS reuses the stale snapshot. Keep the
 * distinguishing entropy within the first ~15 chars.
 */
function shortId(prefix: string): string {
  return `${prefix}${Date.now().toString(36)}${Math.random().toString(36).slice(2, 6)}`
}

describe("HestiaDataView — E2E", () => {
  const container = makeClient()
  const collName = shortId("evb")
  const viewName = shortId("evv")
  const snapName = shortId("evs")
  let docs: HestiaCollection<{ title: string }>
  let view: HestiaDataView<{ title: string }>

  beforeAll(async () => {
    await container.collections.create({ data: { schema: collectionSchema(collName) } })
    docs = container.collection<{ title: string }>(collName)
    await docs.create({ data: { title: "hello" } })
    await docs.create({ data: { title: "other" } })

    view = await HestiaDataView.create(container.client, {
      name: viewName,
      query: {
        target: { name: collName },
        filters: { condition: { field: "title", operator: "eq", value: "hello" } },
      } as any,
    })
  })

  afterAll(async () => {
    await container.collections.delete(collName).catch(() => {})
  })

  it("exposes its name", () => {
    expect(view.name()).toBe(viewName)
    expect(container.view(viewName).name()).toBe(viewName)
  })

  it("finds only rows matching the stored filter", async () => {
    const page = await view.find()
    expect(page.data.length).toBe(1)
    expect(page.data[0]!.title).toBe("hello")
  })

  it("attaches id() and metadata() to every row", async () => {
    const page = await view.find()
    const row = page.data[0]!
    expect(row.id()).toBe(row._id_)
    expect(row.id()).toBeTruthy()
    expect(row.metadata()).toEqual(row._metadata_)
    expect(row.metadata().version).toBeGreaterThanOrEqual(1)
  })

  it("reads a row by id through the view", async () => {
    const id = (await view.find()).data[0]!.id()
    const row = await view.read(id)
    expect(row).toBeDefined()
    expect(row!.id()).toBe(id)
  })

  it("rejects document writes client-side", async () => {
    await expect(view.create({ data: { title: "x" } })).rejects.toThrow(/read-only/)
    await expect(view.update({ id: "id", data: { title: "x" } })).rejects.toThrow(/read-only/)
    await expect(view.delete("id")).rejects.toThrow(/read-only/)
  })

  it("rejects refresh on a virtual view", async () => {
    await expect(view.refresh()).rejects.toThrow()
  })

  it("snapshots and refreshes materialized views", async () => {
    const snap = await HestiaDataView.create(container.client, {
      name: snapName,
      query: { target: { name: collName } } as any,
      materialized: true,
    })
    const firstRead = await snap.find()
    expect(firstRead.data.length).toBe(2)

    await docs.create({ data: { title: "third" } })
    expect((await snap.find()).data.length).toBe(2)

    const descriptor = await snap.refresh()
    expect(descriptor?.name).toBe(snapName)
    expect((await snap.find()).data.length).toBe(3)
  })
})
