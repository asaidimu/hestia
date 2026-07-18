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

  it("login returns access + refresh tokens", async () => {
    const result = await container.auth.login("admin@test.local", "password123")
    expect(result.token.access).toBeTruthy()
    expect(result.token.refresh).toBeTruthy()
    expect(result.token.type).toBe("Bearer")
    expect(result.user.email).toBe("admin@test.local")
    expect(result.user.permissions).toContain("administrator")
  })

  it("login rejects wrong password", async () => {
    await expect(
      container.auth.login("admin@test.local", "wrong-password"),
    ).rejects.toThrow(SystemError)
  })

  it("refresh endpoint accepts JWT refresh token and returns new tokens", async () => {
    const login = await container.auth.login("admin@test.local", "password123")
    const pair = await container.auth.refresh(login.token.refresh)
    expect(pair.access).toBeTruthy()
    expect(pair.refresh).toBeTruthy()
    expect(pair.type).toBe("Bearer")
    // new tokens should differ from original
    expect(pair.access).not.toBe(login.token.access)
    expect(pair.refresh).not.toBe(login.token.refresh)
  })

  it("authenticated collection query works after login", async () => {
    await container.auth.login("admin@test.local", "password123")
    const page = await container.users.find()
    expect(Array.isArray(page.data)).toBe(true)
  })

  it("register a new user as admin", async () => {
    await container.auth.login("admin@test.local", "password123")
    const email = `seq-test-${Date.now()}@example.co`
    const user = await container.auth.register(email, "TestPass1", "Seq User")
    expect(user.email).toBe(email)
    expect(user.name).toBe("Seq User")
    expect(user._id_).toBeTruthy()
  })

  it("auto-refresh: expired access token is refreshed transparently", async () => {
    const login = await container.auth.login("admin@test.local", "password123")

    const garbageAccess = "expired.invalid.token"
    await container.store.set({
      access: garbageAccess,
      refresh: login.token.refresh,
      identity: container.store.get().identity,
    })
    expect(container.store.get().access).toBe(garbageAccess)
    expect(container.store.get().refresh).toBe(login.token.refresh)

    // Spy on the underlying raw client to see what happens
    const rawClient = (container.client as any).raw
    const origPatch = rawClient.patch.bind(rawClient)
    const patchCalls: any[] = []
    rawClient.patch = (...args: any[]) => {
      patchCalls.push(args)
      return origPatch(...args)
    }

    const sendAt = Date.now()
    let page: any
    let caught: any = null
    try {
      page = await container.users.find()
    } catch (err) {
      caught = err
    }

    const elapsed = Date.now() - sendAt
    const finalTokens = container.store.get()

    if (caught) {
      throw new Error(
        `users.find() failed after ${elapsed}ms. ` +
        `err=${caught instanceof Error ? caught.message : JSON.stringify(caught)} ` +
        `refreshCalls=${patchCalls.length} ` +
        `access=${finalTokens.access?.slice(0,30)} refresh=${finalTokens.refresh?.slice(0,30)}`
      )
    }

    if (patchCalls.length === 0) {
      throw new Error(
        `auto-refresh was NOT triggered — server did not return 401/403 ` +
        `for garbage token. Page data received (${page.data.length} items). ` +
        `Tokens after request: access=${finalTokens.access?.slice(0,30)}`
      )
    }

    expect(Array.isArray(page.data)).toBe(true)
    expect(finalTokens.access).toBeTruthy()
    expect(finalTokens.access).not.toBe(garbageAccess)
    expect(finalTokens.refresh).toBeTruthy()
    expect(finalTokens.refresh).not.toBe(login.token.refresh)
  })

  it("logout clears tokens and revokes the access token", async () => {
    await container.auth.login("admin@test.local", "password123")
    await container.auth.logout()
    const state = container.store.get()
    expect(state.access).toBeNull()
    expect(state.refresh).toBeNull()
  })

  it("revoked token is rejected on subsequent requests", async () => {
    const login = await container.auth.login("admin@test.local", "password123")

    // Store tokens then logout (which revokes the access token)
    await container.store.set({
      access: login.token.access,
      refresh: login.token.refresh,
      identity: null,
    })
    await container.auth.logout()

    // Login again to register the session
    const login2 = await container.auth.login("admin@test.local", "password123")

    // Now try to use the revoked old token
    await container.store.set({
      access: login.token.access,
      refresh: login.token.refresh,
      identity: null,
    })

    // Auto-refresh should try the old refresh token, which should also be invalid after refresh
    // If the refresh token was rotated during logout, this should fail
    await expect(container.users.find()).rejects.toThrow()
  })
})
