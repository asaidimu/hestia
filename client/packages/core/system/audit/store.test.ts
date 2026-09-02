import { describe, expect, it } from "vitest"
import { makeClient, uniqueId } from "../../tests/helpers"

describe("HestiaAuditLogs — E2E", () => {
  const container = makeClient()

  it("finds audit log entries", async () => {
    await container.users.list()
    const page = await container.auditLogs.find({ pagination: { type: "offset", offset: 0, limit: 5 } })
    expect(Array.isArray(page.data)).toBe(true)
    expect(page.page).toBeDefined()
  })

  it("lists audit log entries with offset pagination", async () => {
    const page = await container.auditLogs.list()
    expect(Array.isArray(page.data)).toBe(true)
  })

  it("composes the stream URL", () => {
    expect(container.auditLogs.getStreamUrl()).toBe("http://localhost:8070/api/system/audit/log/stream")
  })

  it("returns a live stream controller that can be cancelled", () => {
    const ctl = container.auditLogs.stream({}, () => {})
    expect(ctl.status()).toBe("active")
    ctl.cancel()
    expect(ctl.status()).toBe("cancelled")
  })

  it("rejects mutation operations (append-only)", async () => {
    await expect(container.auditLogs.create({ data: {} as any })).rejects.toThrow()
    await expect(container.auditLogs.update({ data: {}, options: "x" })).rejects.toThrow()
    await expect(container.auditLogs.delete("x")).rejects.toThrow()
    await expect(container.auditLogs.read("x")).rejects.toThrow()
  })
})