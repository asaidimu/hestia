import { describe, expect, it, vi } from "vitest"
import { HestiaClient } from "../container"

const BASE_URL = "http://localhost:8070"

// Note: the test server injects admin claims on every request, so 401
// behavior cannot be tested via e2e. The notifyAuthStateChange flag
// is covered in wails-transport.test.ts via mocked Go bindings.

describe("authenticated() lifecycle", () => {
  it("returns false without a session", async () => {
    const c = new HestiaClient({ baseUrl: BASE_URL })
    expect(await c.authenticated()).toBe(false)
  })

  it("returns true after login", async () => {
    const c = new HestiaClient({ baseUrl: BASE_URL })
    await c.auth.login("admin@test.local", "password123")
    expect(await c.authenticated()).toBe(true)
  })

  it("returns false after logout", async () => {
    const c = new HestiaClient({ baseUrl: BASE_URL })
    await c.auth.login("admin@test.local", "password123")
    await c.auth.logout()
    expect(await c.authenticated()).toBe(false)
  })
})

describe("notifyAuthStateChange flag", () => {
  it("false suppresses callback (authenticated() pure path)", async () => {
    const c = new HestiaClient({ baseUrl: BASE_URL })
    const onAuth = vi.fn()
    c.onAuthStateChange(onAuth)

    const result = await c.authenticated()

    expect(result).toBe(false)
    expect(onAuth).not.toHaveBeenCalled()
  })
})

describe("heartbeat lifecycle", () => {
  it("start/stop does not throw", async () => {
    const c = new HestiaClient({ baseUrl: BASE_URL })
    expect(() => c.startHeartbeat(50)).not.toThrow()
    await new Promise((r) => setTimeout(r, 200))
    expect(() => c.stopHeartbeat()).not.toThrow()
  })
})
