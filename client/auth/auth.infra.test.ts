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
    expect(result.token.access).toBeTruthy()
    expect(result.token.refresh).toBeTruthy()
    expect(result.token.type).toBe("Bearer")
    expect(result.user.email).toBe("admin@test.local")
  })

  it("login rejects wrong password", async () => {
    await expect(
      container.auth.login("admin@test.local", "wrong-password"),
    ).rejects.toThrow(SystemError)
  })

  it("refresh exchanges a refresh token for new tokens", async () => {
    const login = await container.auth.login("admin@test.local", "password123")
    const pair = await container.auth.refresh(login.token.refresh)
    expect(pair.access).toBeTruthy()
    expect(pair.refresh).toBeTruthy()
    expect(pair.type).toBe("Bearer")
  })

  it("auto-refresh on 401: expired access token triggers token refresh", async () => {
    // Log in and get tokens
    const login = await container.auth.login("admin@test.local", "password123")

    // Manually overwrite with an expired-token scenario:
    // Store a deliberately expired access token + the valid refresh token
    // so the next 401 triggers auto-refresh
    await container.store.set({
      access: "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.ZnL4mA", // garbage expired token
      refresh: login.token.refresh,
    })

    // This request should 401, trigger refresh, then succeed
    // The health endpoint is public so it won't 401 — use a collection query instead
    try {
      const page = await container.users.find()
      expect(page.data).toBeDefined()
      // After refresh, tokens should have changed
      const state = container.store.get()
      expect(state.access).not.toBe("eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.ZnL4mA")
      expect(state.refresh).not.toBe(login.token.refresh)
    } catch (err) {
      // If auto-refresh fails entirely, surface details
      const state = container.store.get()
      throw new Error(
        `Auto-refresh failed. access=${state.access?.slice(0, 20)} refresh=${state.refresh?.slice(0, 20)} err=${err instanceof SystemError ? err.code + ": " + err.message : err}`,
      )
    }
  })

  it("logout revokes the current token", async () => {
    const login = await container.auth.login("admin@test.local", "password123")
    // Store the tokens before logout
    await container.store.set({
      access: login.token.access,
      refresh: login.token.refresh,
    })
    await container.auth.logout()
    // Tokens should be cleared
    const state = container.store.get()
    expect(state.access).toBeNull()
    expect(state.refresh).toBeNull()
  })

  it("register a new user as admin", async () => {
    await container.auth.login("admin@test.local", "password123")
    const email = `test-${Date.now()}@example.co`
    const user = await container.auth.register(email, "TestPass1", "Test User")
    expect(user.email).toBe(email)
    expect(user.name).toBe("Test User")
    expect(user._id_).toBeTruthy()
  })
})
