if (typeof (globalThis as any).requestIdleCallback === "undefined") {
  (globalThis as any).requestIdleCallback = (cb: Function) => setTimeout(() => cb({ didTimeout: false }), 0)
}

import { describe, expect, it, beforeAll } from "vitest"
import { HestiaClient } from "../container"
import { SystemError } from "@asaidimu/utils-error"

const BASE_URL = "http://localhost:8070"

let container: HestiaClient

beforeAll(() => {
  container = new HestiaClient({ baseUrl: BASE_URL })
})

describe("health", () => {
  it("reports ok", async () => {
    const health = await container.auth.health()
    expect(health.ok).toBe(true)
    expect(health.bootstrapped).toBe(true)
  })
})

describe("auth", () => {
  it("logs in as admin", async () => {
    const result = await container.auth.login("admin@test.local", "password123")
    expect(result.token.access).toBeTruthy()
    expect(result.token.refresh).toBeTruthy()
    expect(result.token.type).toBe("Bearer")
    expect(result.user.email).toBe("admin@test.local")
    expect(result.user.permissions).toContain("administrator")
  })

  it("rejects wrong password", async () => {
    await expect(
      container.auth.login("admin@test.local", "wrong-password"),
    ).rejects.toThrow(SystemError)
  })

  it("registers a new user", async () => {
    // need admin login first for the token
    await container.auth.login("admin@test.local", "password123")
    const user = await container.auth.register("newuser@test.local", "TestPass1", "New User")
    expect(user.email).toBe("newuser@test.local")
    expect(user.name).toBe("New User")
    expect(user._id_).toBeTruthy()
  })

  it("refreshes a token", async () => {
    const loginResult = await container.auth.login("admin@test.local", "password123")
    const token = await container.auth.refresh(loginResult.token.refresh)
    expect(token.access).toBeTruthy()
    expect(token.refresh).toBeTruthy()
  })
})

describe("users collection (_user_)", () => {
  let registeredId: string

  beforeAll(async () => {
    await container.auth.login("admin@test.local", "password123")
    const user = await container.auth.register("usertest@test.local", "TestPass1", "User Test")
    registeredId = user._id_
  })

  it("lists users via collection query", async () => {
    const page = await container.users.find()
    expect(page.data.length).toBeGreaterThanOrEqual(2)
    const found = page.data.find((u) => u.email === "admin@test.local")
    expect(found).toBeTruthy()
  })

  it("gets a user by id", async () => {
    const doc = await container.users.read(registeredId)
    expect(doc).toBeDefined()
    expect(doc!.email).toBe("usertest@test.local")
    expect(doc!.name).toBe("User Test")
  })

  it("updates a user", async () => {
    const updated = await container.users.update({ data: { name: "Updated Name" }, options: registeredId })
    expect(updated!.name).toBe("Updated Name")
  })

  it("changes a user password", async () => {
    await container.users.changePassword(registeredId, "TestPass1", "NewPass1")
    const loginResult = await container.auth.login("usertest@test.local", "NewPass1")
    expect(loginResult.token.access).toBeTruthy()
  })
})

describe("api keys (_api_key_)", () => {
  let keyId: string
  let keySecret: string

  beforeAll(async () => {
    await container.auth.login("admin@test.local", "password123")
  })

  it("creates an api key", async () => {
    const key = await container.keys.create({ data: { name: "Test Key" } })
    expect(key!.name).toBe("Test Key")
    expect((key as any).key).toBeTruthy()
    expect((key as any).prefix).toBeTruthy()
    keyId = key!._id_
    keySecret = (key as any).key
  })

  it("lists api keys", async () => {
    const page = await container.keys.list()
    expect(page.data.length).toBeGreaterThanOrEqual(1)
    expect(page.data.some((k) => k._id_ === keyId)).toBe(true)
  })

  it("gets an api key", async () => {
    const doc = await container.keys.read(keyId)
    expect(doc).toBeDefined()
    expect(doc!._id_).toBe(keyId)
    expect(doc!.name).toBe("Test Key")
  })

  it("updates an api key", async () => {
    const updated = await container.keys.update({ data: { name: "Renamed Key" }, options: keyId })
    expect(updated!.name).toBe("Renamed Key")
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
  }, 12000)
})

describe("pagination via users collection", () => {
  beforeAll(async () => {
    await container.auth.login("admin@test.local", "password123")
  })

  it("provides a reactive pager", async () => {
    const pager = container.users.page()
    const initial = pager.page()
    expect(initial.loading).toBe(false)
    expect(initial.page.number).toBe(1)
    expect(initial.page.size).toBe(20)

    await pager.resize(5, 1)

    const loading = pager.page()
    expect(loading.loading).toBe(true)
    // page metadata reflects the last known state until fetch completes
    expect(loading.page.number).toBe(1)

    // subscribe starts empty and delivers on navigate
    const values: any[] = []
    const unsub = pager.subscribe((p) => values.push(p))
    expect(values.length).toBe(0)

    // navigate triggers debounce(50ms) + 2×requestIdleCallback → fetch
    await pager.navigate(2)
    await new Promise<void>((resolve) => setTimeout(resolve, 100))
    expect(values.length).toBeGreaterThanOrEqual(1)

    unsub()
  })
})
