import { describe, expect, it, vi, beforeEach } from "vitest"
import { HestiaAuth } from "./store"
import { HttpTransport, type IdentityProvider } from "../core/client"
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
  let identity: any = null
  return {
    identity: () => identity,
    setIdentity: vi.fn(async (id: any) => {
      identity = id
    }),
    clear: vi.fn(async () => {
      identity = null
    }),
  }
}

function okResponse<T>(data: T): ApiResponse<T> {
  return { success: true, status: 200, data, raw: new Response(), headers: new Headers() }
}

function errorResponse(status: number): ApiResponse<never> {
  return { success: false, status, data: undefined as never, raw: new Response(null, { status }), headers: new Headers() }
}

function mockDocsResponse() {
  return okResponse({
    data: [
      { name: "system:auth:session:create", http: { method: "POST", route: "/system/auth/session" }, input: {} },
      { name: "system:auth:session:delete", http: { method: "DELETE", route: "/system/auth/session" }, input: {} },
    ],
  })
}

describe("HestiaAuth login", () => {
  let provider: IdentityProvider
  let client: HttpTransport
  let auth: HestiaAuth
  let raw: any

  beforeEach(() => {
    provider = makeProvider()
    client = new HttpTransport("http://test.local", "/api", provider)
    auth = new HestiaAuth(client, provider)
    raw = (createNetworkClient as ReturnType<typeof vi.fn>).mock.results[0]
      ?.value
    if (!raw) {
      const mock = (createNetworkClient as ReturnType<typeof vi.fn>).mock
      raw = mock.results[mock.results.length - 1]!.value
    }
    vi.clearAllMocks()
  })

  it("login stores identity", async () => {
    raw.get.mockResolvedValueOnce(mockDocsResponse())
    raw.post.mockResolvedValueOnce(
      okResponse({
        data: {
          user: { _id_: "u1", email: "a@b.co", name: "A", permissions: ["administrator"] },
        },
      }),
    )

    const result = await auth.login("a@b.co", "pwd")
    expect(result.user.email).toBe("a@b.co")
    expect(provider.identity()).toEqual({ _id_: "u1", email: "a@b.co", name: "A", permissions: ["administrator"] })
  })

  it("logout clears identity", async () => {
    raw.get.mockResolvedValueOnce(mockDocsResponse())
    raw.post.mockResolvedValueOnce(
      okResponse({
        data: {
          user: { _id_: "u1", email: "a@b.co", name: "A", permissions: [] },
        },
      }),
    )
    raw.get.mockResolvedValueOnce(mockDocsResponse())
    raw.delete.mockResolvedValueOnce(okResponse({}))

    await auth.login("a@b.co", "pwd")
    expect(provider.identity()).toBeTruthy()

    await auth.logout()
    expect(provider.identity()).toBeNull()
  })

  it("login rejects wrong password", async () => {
    raw.get.mockResolvedValueOnce(mockDocsResponse())
    raw.post.mockResolvedValueOnce(errorResponse(401))

    await expect(auth.login("a@b.co", "wrong")).rejects.toThrow()
    expect(provider.identity()).toBeNull()
  })
})
