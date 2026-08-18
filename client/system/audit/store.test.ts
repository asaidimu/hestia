import { describe, expect, it } from "vitest"
import { makeClient, uniqueId } from "../../tests/helpers"

describe("HestiaLogs — E2E", () => {
  const container = makeClient()

  it("finds audit log entries", async () => {
    await container.users.list()
    const page = await container.logs.find({ pagination: { type: "offset", offset: 0, limit: 5 } })
    expect(Array.isArray(page.data)).toBe(true)
    expect(page.page).toBeDefined()
  })

  it("lists audit log entries with offset pagination", async () => {
    const page = await container.logs.list()
    expect(Array.isArray(page.data)).toBe(true)
  })

  it("composes the stream URL", () => {
    expect(container.logs.getStreamUrl()).toBe("http://localhost:8070/api/system/audit/log/stream")
  })

  it("returns a live stream controller that can be cancelled", () => {
    const ctl = container.logs.stream({}, () => {})
    expect(ctl.status()).toBe("active")
    ctl.cancel()
    expect(ctl.status()).toBe("cancelled")
  })

  it("rejects mutation operations (append-only)", async () => {
    await expect(container.logs.create({ data: {} as any })).rejects.toThrow()
    await expect(container.logs.update({ data: {}, options: "x" })).rejects.toThrow()
    await expect(container.logs.delete("x")).rejects.toThrow()
    await expect(container.logs.read("x")).rejects.toThrow()
  })
})