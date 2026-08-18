import { describe, expect, it } from "vitest"
import { SystemError } from "@asaidimu/utils-error"
import { makeClient, uniqueId } from "../../tests/helpers"

// E2E against the test server (auth disabled, admin claims injected).
// Password-reset / bootstrap flows are excluded: password reset needs an SMTP
// sink and bootstrap would mutate the shared admin password.

describe("HestiaAuth — E2E", () => {
  it("health check reports ok", async () => {
    const client = makeClient()
    const health = await client.auth.health()
    expect(health.ok).toBe(true)
    expect(health.bootstrapped).toBe(true)
  })

  it("login stores identity", async () => {
    const client = makeClient()
    const result = await client.auth.login("admin@test.local", "password123")
    expect(result.user.email).toBe("admin@test.local")
    expect(await client.authenticated()).toBe(true)
  })

  it("logout clears identity", async () => {
    const client = makeClient()
    await client.auth.login("admin@test.local", "password123")
    expect(await client.authenticated()).toBe(true)
    await client.auth.logout()
    expect(await client.authenticated()).toBe(false)
  })

  it("login rejects wrong password", async () => {
    const client = makeClient()
    await expect(client.auth.login("admin@test.local", "wrong")).rejects.toThrow(SystemError)
    expect(await client.authenticated()).toBe(false)
  })

  it("register creates a user", async () => {
    const client = makeClient()
    const email = uniqueId("auth-e2e") + "@example.co"
    const user = await client.auth.register(email, "TestPass1", "Auth E2E")
    expect(user.email).toBe(email)
    expect(user.name).toBe("Auth E2E")
    expect(user._id_).toBeTruthy()
  })
})