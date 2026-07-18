import { describe, expect, it, vi, beforeEach } from "vitest"
import { HestiaAuth } from "./store"
import { HestiaNetworkClient, type IdentityProvider } from "../core/client"
import type { ApiResponse } from "@asaidimu/network-client"

vi.mock("@asaidimu/network-client", () => {
  const mockRaw = {
    get: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  }
  return {
    createNetworkClient: vi.fn(() => mockRaw),
  }
})

import { createNetworkClient } from "@asaidimu/network-client"

function makeProvider(): IdentityProvider {
  let state: { access: string | null; refresh: string | null } = {
    access: null,
    refresh: null,
  }
  return {
    identity: () => null,
    token: (k: "access" | "refresh") => state[k],
    setTokens: vi.fn(async (a: string, r: string) => {
      state = { access: a, refresh: r }
    }),
    setIdentity: vi.fn(),
    clear: vi.fn(async () => {
      state = { access: null, refresh: null }
    }),
  }
}

function okResponse<T>(data: T): ApiResponse<T> {
  return { success: true, status: 200, data, raw: new Response(), headers: new Headers() }
}

function errorResponse(status: number): ApiResponse<never> {
  return { success: false, status, data: undefined as never, raw: new Response(null, { status }), headers: new Headers() }
}

describe("HestiaAuth refresh sequence", () => {
  let provider: IdentityProvider
  let client: HestiaNetworkClient
  let auth: HestiaAuth
  let raw: any

  beforeEach(() => {
    provider = makeProvider()
    client = new HestiaNetworkClient("http://test.local", "/api", provider)
    auth = new HestiaAuth(client, provider)
    raw = (createNetworkClient as ReturnType<typeof vi.fn>).mock.results[0]
      ?.value
    if (!raw) {
      const mock = (createNetworkClient as ReturnType<typeof vi.fn>).mock
      raw = mock.results[mock.results.length - 1]!.value
    }
    vi.clearAllMocks()
  })

  it("login stores tokens and calls refresh endpoint", async () => {
    raw.post.mockResolvedValueOnce(
      okResponse({
        data: {
          token: {
            access: "access-token-1",
            refresh: "refresh-token-1",
            type: "Bearer",
            validity: 900,
          },
          user: { _id_: "u1", email: "a@b.co", name: "A", permissions: ["administrator"] },
        },
      }),
    )

    const result = await auth.login("a@b.co", "pwd")
    expect(result.token.access).toBe("access-token-1")
    expect(result.token.refresh).toBe("refresh-token-1")

    expect(raw.post).toHaveBeenCalledWith(
      "api/system/auth/session",
      { email: "a@b.co", password: "pwd" },
      {},
      undefined,
    )
    // tokens stored
    expect(provider.token("access")).toBe("access-token-1")
    expect(provider.token("refresh")).toBe("refresh-token-1")
  })

  it("refresh() exchanges a refresh token for new tokens", async () => {
    raw.patch.mockResolvedValueOnce(
      okResponse({
        data: {
          token: {
            access: "access-token-2",
            refresh: "refresh-token-2",
            type: "Bearer",
            validity: 900,
          },
        },
      }),
    )

    const pair = await auth.refresh("old-refresh-token")
    expect(pair.access).toBe("access-token-2")
    expect(pair.refresh).toBe("refresh-token-2")
    expect(raw.patch).toHaveBeenCalledWith(
      "api/system/auth/session",
      { refresh_token: "old-refresh-token" },
      {},
      undefined,
    )
  })
})

describe("HestiaNetworkClient auto-refresh", () => {
  let provider: IdentityProvider
  let client: HestiaNetworkClient
  let raw: any

  function initClient(onAuthChanged?: () => void) {
    provider = makeProvider()
    client = new HestiaNetworkClient("http://test.local", "/api", provider, onAuthChanged)
    const mock = (createNetworkClient as ReturnType<typeof vi.fn>).mock
    raw = mock.results[mock.results.length - 1]!.value
    vi.clearAllMocks()
  }

  it("auto-refreshes on 401 and retries the original request", async () => {
    initClient()

    // store initial tokens
    await provider.setTokens("expired-access", "valid-refresh")

    // First call fails with 401, refresh succeeds, retry succeeds
    raw.get.mockResolvedValueOnce(errorResponse(401))
    raw.patch.mockResolvedValueOnce(
      okResponse({
        data: {
          token: { access: "new-access", refresh: "new-refresh", type: "Bearer", validity: 900 },
        },
      }),
    )
    raw.get.mockResolvedValueOnce(
      okResponse({ data: [{ _id_: "d1" }] }),
    )

    const res = await client.get<{ data: any[] }>("/collection/items")

    expect(res.data).toEqual({ data: [{ _id_: "d1" }] })
    // refresh endpoint was called
    expect(raw.patch).toHaveBeenCalledWith(
      "api/system/auth/session",
      { refresh_token: "valid-refresh" },
    )
    // new tokens stored
    expect(provider.token("access")).toBe("new-access")
    expect(provider.token("refresh")).toBe("new-refresh")
    // original GET was retried
    expect(raw.get).toHaveBeenCalledTimes(2)
  })

  it("does NOT auto-refresh on auth endpoints", async () => {
    initClient()

    await provider.setTokens("expired-access", "valid-refresh")

    raw.post.mockResolvedValueOnce(errorResponse(401))

    await expect(
      client.post("/system/auth/session", { email: "a", password: "b" }),
    ).rejects.toThrow()

    // refresh should NOT have been called
    expect(raw.patch).not.toHaveBeenCalled()
  })

  it("calls onAuthStateChanged after refresh", async () => {
    const onChanged = vi.fn()
    initClient(onChanged)

    await provider.setTokens("expired-access", "valid-refresh")

    raw.get.mockResolvedValueOnce(errorResponse(401))
    raw.patch.mockResolvedValueOnce(
      okResponse({
        data: {
          token: { access: "new-access", refresh: "new-refresh", type: "Bearer", validity: 900 },
        },
      }),
    )
    raw.get.mockResolvedValueOnce(okResponse({ data: [] }))

    await client.get("/items")
    expect(onChanged).toHaveBeenCalledTimes(1)
  })

  it("clears tokens and throws when refresh also fails", async () => {
    initClient()

    await provider.setTokens("expired-access", "bad-refresh")

    raw.get.mockResolvedValueOnce(errorResponse(401))
    raw.patch.mockResolvedValueOnce(errorResponse(403))

    await expect(client.get("/items")).rejects.toThrow()
    // tokens cleared
    expect(provider.token("access")).toBeNull()
    expect(provider.token("refresh")).toBeNull()
  })

  it("deduplicates concurrent refresh calls", async () => {
    initClient()

    await provider.setTokens("expired-access", "valid-refresh")

    // both requests fail with 401
    raw.get.mockResolvedValueOnce(errorResponse(401))
    raw.get.mockResolvedValueOnce(errorResponse(401))
    raw.patch.mockResolvedValueOnce(
      okResponse({
        data: {
          token: { access: "new-access", refresh: "new-refresh", type: "Bearer", validity: 900 },
        },
      }),
    )
    // after refresh, both retries succeed
    raw.get.mockResolvedValue(okResponse({ data: [] }))

    const [r1, r2] = await Promise.all([
      client.get("/a"),
      client.get("/b"),
    ])

    expect(r1.status).toBe(200)
    expect(r2.status).toBe(200)
    // refresh should only be called ONCE
    expect(raw.patch).toHaveBeenCalledTimes(1)
  })
})
