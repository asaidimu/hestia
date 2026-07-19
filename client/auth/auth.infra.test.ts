import { describe, expect, it } from "vitest"
import { HestiaClient } from "../container"
import { SystemError } from "@asaidimu/utils-error"

const BASE_URL = "http://localhost:8070"

describe("auth sequence against real server", () => {
  const container = new HestiaClient({ baseUrl: BASE_URL })

  it("health check returns ok", async () => {
    const health = await container.auth.health()
    expect(health.ok).toBe(true)
    expect(typeof health.bootstrapped).toBe("boolean")
  })

  it("login succeeds with valid credentials", async () => {
    const result = await container.auth.login("admin@test.local", "password123")
    expect(result.user.email).toBe("admin@test.local")
  })

  it("login rejects wrong password", async () => {
    await expect(
      container.auth.login("admin@test.local", "wrong-password"),
    ).rejects.toThrow(SystemError)
  })

  it("register a new user as admin (no login needed)", async () => {
    const email = `test-${Date.now()}@example.co`
    const user = await container.auth.register(email, "TestPass1", "Test User")
    expect(user.email).toBe(email)
    expect(user.name).toBe("Test User")
    expect(user._id_).toBeTruthy()
  })

  it("collection queries work without auth", async () => {
    const page = await container.users.find()
    expect(Array.isArray(page.data)).toBe(true)
  })
})
