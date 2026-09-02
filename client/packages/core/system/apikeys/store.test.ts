import { describe, expect, it } from "vitest"
import { makeClient, uniqueId } from "../../tests/helpers"

describe("HestiaKeyStore — E2E", () => {
  const container = makeClient()
  const keyName = uniqueId("key-e2e")
  let keyId: string
  let keySecret: string

  it("creates an api key", async () => {
    const key = await container.keys.create({ data: { name: keyName } })
    expect(key).toBeDefined()
    expect(key!._id_).toBeTruthy()
    expect((key as any).key).toBeTruthy()
    keyId = key!._id_
    keySecret = (key as any).key
  })

  it("lists api keys", async () => {
    const page = await container.keys.list()
    expect(page.data.some((k) => k._id_ === keyId)).toBe(true)
  })

  it("gets an api key by id", async () => {
    const key = await container.keys.read(keyId)
    expect(key).toBeDefined()
    expect(key!._id_).toBe(keyId)
    expect(key!.name).toBe(keyName)
  })

  it("returns undefined for a missing key", async () => {
    await expect(container.keys.read("nonexistent-key")).resolves.toBeUndefined()
  })

  it("updates an api key", async () => {
    const updated = await container.keys.update({ data: { name: `${keyName}-renamed` }, options: keyId })
    expect(updated!._id_).toBe(keyId)
    expect(updated!.name).toBe(`${keyName}-renamed`)
  })

  it("rotates an api key", async () => {
    const rotated = await container.keys.rotate(keyId)
    expect(rotated._id_).toBe(keyId)
    expect((rotated as any).key).toBeTruthy()
    expect((rotated as any).key).not.toBe(keySecret)
  })

  it("deletes an api key", async () => {
    await container.keys.delete(keyId)
    await expect(container.keys.read(keyId)).resolves.toBeUndefined()
  })
})