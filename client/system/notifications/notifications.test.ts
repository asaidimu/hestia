import { describe, expect, it, vi, beforeEach } from "vitest"
import { HestiaNotificationStore } from "./store"
import { HttpTransport, type IdentityProvider } from "../../core/client"
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
  return {
    identity: () => null,
    setIdentity: vi.fn(),
    clear: vi.fn(),
  }
}

function okResponse<T>(data: T): ApiResponse<T> {
  return { success: true, status: 200, data, raw: new Response(), headers: new Headers() }
}

describe("HestiaNotificationStore", () => {
  let store: HestiaNotificationStore
  let raw: any

  beforeEach(() => {
    const client = new HttpTransport("http://test.local", "/api")
    const mock = (createNetworkClient as ReturnType<typeof vi.fn>).mock
    raw = mock.results[mock.results.length - 1]!.value
    vi.clearAllMocks()
    store = new HestiaNotificationStore(client)
  })

  describe("list", () => {
    it("returns notification documents", async () => {
      raw.get.mockResolvedValueOnce(
        okResponse({
          data: [
            {
              _id_: "n1",
              user_id: "u1",
              type: "password_reset",
              subject: "Reset your password",
              body: "Click here to reset",
              read: false,
              created_at: 1000,
              _metadata_: {},
            },
          ],
        }),
      )

      const result = await store.list()
      expect(result).toHaveLength(1)
      expect(result[0].type).toBe("password_reset")
      expect(result[0].read).toBe(false)
    })

    it("returns empty array when no data", async () => {
      raw.get.mockResolvedValueOnce(okResponse({}))
      const result = await store.list()
      expect(result).toEqual([])
    })
  })

  describe("markRead", () => {
    it("resolves without error on success", async () => {
      raw.patch.mockResolvedValueOnce(okResponse({}))
      await expect(store.markRead("n1")).resolves.toBeUndefined()
    })
  })

  describe("markAllRead", () => {
    it("resolves without error on success", async () => {
      raw.patch.mockResolvedValueOnce(okResponse({}))
      await expect(store.markAllRead()).resolves.toBeUndefined()
    })
  })

  describe("countUnread", () => {
    it("returns count from response", async () => {
      raw.get.mockResolvedValueOnce(
        okResponse({
          data: { count: 3 },
        }),
      )

      const count = await store.countUnread()
      expect(count).toBe(3)
    })

    it("returns 0 when response has no data", async () => {
      raw.get.mockResolvedValueOnce(okResponse({}))
      const count = await store.countUnread()
      expect(count).toBe(0)
    })
  })
})
