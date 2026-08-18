import { describe, expect, it, beforeAll, afterAll } from "vitest"
import { makeClient, uniqueId, collectionSchema } from "../../tests/helpers"

describe("HestiaCollections — E2E", () => {
  const container = makeClient()
  const collName = uniqueId("e2e_coll")

  afterAll(async () => {
    await container.collections.delete(collName).catch(() => {})
  })

  it("creates a collection (name derived from schema, not required)", async () => {
    const created = await container.collections.create({ data: { schema: collectionSchema(collName) } })
    expect(created).toBeDefined()
    expect(created!.name).toBe(collName)
    expect(created!._id_).toBeTruthy()
    expect(created!._metadata_).toBeDefined()
  })

  it("lists collections (find)", async () => {
    const page = await container.collections.find()
    expect(page.data.some((c) => c.name === collName)).toBe(true)
    const coll = page.data.find((c) => c.name === collName)
    expect(coll?._id_).toBeTruthy()
    expect(coll?._metadata_).toBeDefined()
  })

  it("gets a collection by name", async () => {
    const doc = await container.collections.read(collName)
    expect(doc).toBeDefined()
    expect(doc!.name).toBe(collName)
    expect(doc!._id_).toBeTruthy()
    expect(doc!._metadata_).toBeDefined()
  })

  it("deletes a collection", async () => {
    await container.collections.delete(collName)
    const page = await container.collections.find()
    expect(page.data.some((c) => c.name === collName)).toBe(false)
  })

  it("documents() returns a bound document store", () => {
    const docs = container.collections.documents<any>("_user_")
    expect(docs.name()).toBe("_user_")
  })
})