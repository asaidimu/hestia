import { describe, expect, it, beforeAll, afterAll } from "vitest"
import { makeClient, uniqueId, collectionSchema } from "../tests/helpers"
import type { HestiaCollection } from "./collection"

describe("HestiaCollection (document CRUD) — E2E", () => {
  const container = makeClient()
  const collName = uniqueId("e2e_docs")
  let docs: HestiaCollection<{ title: string }>
  let docId: string

  beforeAll(async () => {
    await container.collections.create({ data: { schema: collectionSchema(collName) } })
    docs = container.collection<{ title: string }>(collName)
  })

  afterAll(async () => {
    await container.collections.delete(collName).catch(() => {})
  })

  it("creates a document", async () => {
    const doc = await docs.create({ data: { title: "hello" } })
    expect(doc).toBeDefined()
    expect(doc!._id_).toBeTruthy()
    expect(doc!.title).toBe("hello")
    docId = doc!._id_
  })

  it("strips server-managed envelope keys from creates", async () => {
    const doc = await docs.create({ data: { title: "forged", _id_: "forged-id", _metadata_: {} } as any })
    expect(doc).toBeDefined()
    expect(doc!.id()).not.toBe("forged-id")
    expect(doc!.title).toBe("forged")
  })

  it("queries documents", async () => {
    const page = await docs.find()
    expect(page.data.length).toBeGreaterThanOrEqual(1)
    expect(page.data.some((d) => d._id_ === docId)).toBe(true)
  })

  it("lists documents with offset pagination", async () => {
    const page = await docs.list()
    expect(Array.isArray(page.data)).toBe(true)
    expect(page.page).toBeDefined()
  })

  it("gets a document by id", async () => {
    const doc = await docs.read(docId)
    expect(doc).toBeDefined()
    expect(doc!._id_).toBe(docId)
    expect(doc!.title).toBe("hello")
  })

  it("updates a document", async () => {
    const updated = await docs.update({ id: docId, data: { title: "bye" } })
    expect(updated!._id_).toBe(docId)
    expect(updated!.title).toBe("bye")
  })

  it("updates documents by filter", async () => {
    await docs.create({ data: { title: "batch-a" } })
    await docs.create({ data: { title: "batch-b" } })
    const updated = await docs.update({
      data: { title: "batched" },
      filter: { condition: { field: "title", operator: "eq", value: "batch-a" } } as any,
    })
    expect(updated).toBeDefined()
    expect(updated!.title).toBe("batched")
    const page = await docs.find()
    expect(page.data.some((d) => d.title === "batch-b")).toBe(true)
  })

  it("rejects update with neither id nor filter", async () => {
    await expect(docs.update({ data: { title: "x" } })).rejects.toThrow(/exactly one of id or filter/)
  })

  it("deletes a document", async () => {
    await docs.delete(docId)
    const page = await docs.find()
    expect(page.data.some((d) => d._id_ === docId)).toBe(false)
  })
})