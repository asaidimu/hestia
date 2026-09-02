import { describe, expect, it } from "vitest"
import { makeClient } from "../../tests/helpers"
import { HestiaOperations } from "./store"

describe("HestiaOperations — E2E", () => {
  const operations = new HestiaOperations(makeClient().client)

  it("lists policy bindings", async () => {
    const page = await operations.list()
    expect(page.data.length).toBeGreaterThan(0)
    expect(page.data.some((b) => b.name === "system:auth:session:create")).toBe(true)
  })

  it("gets a binding by name", async () => {
    const doc = await operations.read("system:auth:session:create")
    expect(doc).toBeDefined()
    expect(doc!.name).toBe("system:auth:session:create")
    expect(doc!._id_).toBeTruthy()
    expect(doc!._metadata_).toBeDefined()
  })
})