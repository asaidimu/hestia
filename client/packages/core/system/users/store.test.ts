import { describe, expect, it, beforeAll, afterAll } from "vitest"
import { makeClient, uniqueId } from "../../tests/helpers"

describe("HestiaUsers — E2E", () => {
  const container = makeClient()
  const email = uniqueId("user-e2e") + "@example.co"
  const password = "Passw0rd1"
  let userId: string

  afterAll(async () => {
    await container.users.delete(userId).catch(() => {})
  })

  it("creates a user", async () => {
    const user = await container.users.create({ data: { email, password, name: "E2E User" } })
    expect(user).toBeDefined()
    expect(user!._id_).toBeTruthy()
    expect(user!.email).toBe(email)
    userId = user!._id_
  })

  it("lists users via collection query (find)", async () => {
    const page = await container.users.find()
    expect(page.data.some((u) => u._id_ === userId)).toBe(true)
  })

  it("lists users with offset pagination", async () => {
    const page = await container.users.list()
    expect(Array.isArray(page.data)).toBe(true)
    expect(page.page).toBeDefined()
  })

  it("gets a user by id", async () => {
    const user = await container.users.read(userId)
    expect(user).toBeDefined()
    expect(user!._id_).toBe(userId)
    expect(user!.email).toBe(email)
  })

  it("updates a user", async () => {
    const updated = await container.users.update({ data: { name: "Renamed User" }, options: userId })
    expect(updated!._id_).toBe(userId)
    expect(updated!.name).toBe("Renamed User")
  })

  it("changes a user password", async () => {
    await container.users.changePassword(userId, password, "NewPass1")
    const login = await container.auth.login(email, "NewPass1")
    expect(login.user.email).toBe(email)
  })

  it("deletes a user", async () => {
    await container.users.delete(userId)
  })
})