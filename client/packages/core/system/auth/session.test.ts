import { describe, expect, it, vi } from "vitest"
import { SystemError } from "@asaidimu/utils-error"
import type { IdentityProvider, Transport } from "../../core/client"
import { HestiaAuth } from "./store"
import type { UserIdentity } from "../users/types"

const USER = {
  _id_: "user-1",
  email: "test@example.com",
  name: "Test",
} as unknown as UserIdentity

function makeProvider() {
  let identity: UserIdentity | null = null
  const provider: IdentityProvider = {
    identity: () => identity,
    setIdentity: vi.fn(async (id) => {
      identity = id
    }),
    clear: vi.fn(async () => {
      identity = null
    }),
  }
  return { provider, current: () => identity }
}

function makeAuth(dispatch: (name: string, input?: unknown) => Promise<unknown>) {
  const { provider, current } = makeProvider()
  const client = { dispatch } as unknown as Transport
  return { auth: new HestiaAuth(client, provider), provider, current }
}

describe("HestiaAuth.session", () => {
  it("adopts the server session without triggering the global 401 handler", async () => {
    const dispatch = vi.fn(async () => ({
      data: { data: { user: USER, expires_at: 1_800_000_000 } },
      status: 200,
    }))
    const { auth, current } = makeAuth(dispatch)

    const result = await auth.session()

    expect(dispatch).toHaveBeenCalledWith("system:auth:session:get", {
      notifyAuthStateChange: false,
    })
    expect(result).toEqual({ user: USER, expires_at: 1_800_000_000 })
    expect(current()).toBe(USER)
  })

  it("returns null and clears the identity on 401", async () => {
    const { auth, provider, current } = makeAuth(async () => {
      throw new SystemError({ code: "UNAUTHORIZED", message: "no active session" })
    })
    await provider.setIdentity(USER)

    await expect(auth.session()).resolves.toBeNull()
    expect(current()).toBeNull()
  })

  it("rethrows non-auth failures instead of logging the user out", async () => {
    const { auth, provider, current } = makeAuth(async () => {
      throw new SystemError({ code: "NETWORK", message: "offline" })
    })
    await provider.setIdentity(USER)

    await expect(auth.session()).rejects.toMatchObject({ code: "NETWORK" })
    expect(current()).toBe(USER)
  })
})
