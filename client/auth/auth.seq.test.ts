import { describe, expect, it, beforeAll } from "vitest"
import { HestiaClient } from "../container"
import { SystemError } from "@asaidimu/utils-error"

const BASE_URL = "http://localhost:8070"

let container: HestiaClient

beforeAll(() => {
  container = new HestiaClient({ baseUrl: BASE_URL })
})

describe("auth sequence", () => {
  it("health check works without auth (public endpoint)", async () => {
    const health = await container.auth.health()
    expect(health.ok).toBe(true)
    expect(health.bootstrapped).toBe(true)
  })

  it("login returns user identity", async () => {
    const result = await container.auth.login("admin@test.local", "password123")
    expect(result.user.email).toBe("admin@test.local")
    expect(result.user.permissions).toContain("administrator")
  })

  it("login rejects wrong password", async () => {
    await expect(
      container.auth.login("admin@test.local", "wrong-password"),
    ).rejects.toThrow(SystemError)
  })

  it("collection query works without login (auth disabled)", async () => {
    const page = await container.users.find()
    expect(Array.isArray(page.data)).toBe(true)
  })

  it("register a new user as admin (no login needed)", async () => {
    const email = `seq-test-${Date.now()}@example.co`
    const user = await container.auth.register(email, "TestPass1", "Seq User")
    expect(user.email).toBe(email)
    expect(user.name).toBe("Seq User")
    expect(user._id_).toBeTruthy()
  })

  it("logout is a no-op but does not error", async () => {
    await container.auth.login("admin@test.local", "password123")
    await container.auth.logout()
  })
})
