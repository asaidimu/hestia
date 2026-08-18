import { describe, expect, it, afterAll } from "vitest"
import { makeClient, uniqueId } from "../../tests/helpers"

describe("HestiaSettingStore — E2E", () => {
  const container = makeClient()
  const key = uniqueId("setting-e2e")
  const value = { mode: "dark", retries: 3 }

  afterAll(async () => {
    await container.settings.delete(key).catch(() => {})
  })

  it("sets a setting", async () => {
    await container.settings.set(key, value)
  })

  it("gets a setting by key", async () => {
    const doc = await container.settings.get(key)
    expect(doc).toBeDefined()
    expect(doc!.key).toBe(key)
    expect(doc!.value).toEqual(value)
  })

  it("lists settings", async () => {
    const docs = await container.settings.list()
    expect(docs.some((d) => d.key === key)).toBe(true)
  })

  it("deletes a setting", async () => {
    await container.settings.delete(key)
    const doc = await container.settings.get(key)
    expect(doc).toBeUndefined()
  })
})