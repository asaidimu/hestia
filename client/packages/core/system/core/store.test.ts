import { describe, expect, it } from "vitest"
import { makeClient } from "../../tests/helpers"

describe("HestiaCore — E2E", () => {
  const container = makeClient()

  it("lists the core docs (capabilities)", async () => {
    const page = await container.core.find()
    expect(page.data.length).toBeGreaterThan(0)
    expect(page.data.some((d) => d.name === "system:auth:session:create")).toBe(true)
  })

  it("list delegates to find", async () => {
    const page = await container.core.list()
    expect(page.data.length).toBeGreaterThan(0)
  })

  it("rejects mutation operations (read-only)", async () => {
    await expect(container.core.create({ data: {} })).rejects.toThrow()
    await expect(container.core.update({ data: {}, options: "x" })).rejects.toThrow()
    await expect(container.core.delete("x")).rejects.toThrow()
    await expect(container.core.read("x")).rejects.toThrow()
  })
})