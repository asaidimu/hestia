import { describe, expect, it, vi, beforeEach } from "vitest"
import { HestiaSettingStore } from "./store"
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

describe("HestiaSettingStore", () => {
  let store: HestiaSettingStore
  let raw: any

  beforeEach(() => {
    const client = new HttpTransport("http://test.local", "/api")
    const mock = (createNetworkClient as ReturnType<typeof vi.fn>).mock
    raw = mock.results[mock.results.length - 1]!.value
    vi.clearAllMocks()
    store = new HestiaSettingStore(client)
  })

  describe("list", () => {
    it("returns setting documents", async () => {
      raw.get.mockResolvedValueOnce(
        okResponse({
          data: [
            {
              _id_: "s1",
              key: "theme",
              value: { mode: "dark" },
              _metadata_: {},
            },
          ],
        }),
      )

      const result = await store.list()
      expect(result).toHaveLength(1)
      expect(result[0].key).toBe("theme")
    })

    it("returns empty array when no data", async () => {
      raw.get.mockResolvedValueOnce(okResponse({}))
      const result = await store.list()
      expect(result).toEqual([])
    })
  })

  describe("get", () => {
    it("returns the setting document", async () => {
      raw.get.mockResolvedValueOnce(
        okResponse({
          data: {
            _id_: "s1",
            key: "theme",
            value: { mode: "dark" },
            _metadata_: {},
          },
        }),
      )

      const result = await store.get("theme")
      expect(result).toBeDefined()
      expect(result!.key).toBe("theme")
      expect(result!.value).toEqual({ mode: "dark" })
    })

    it("returns undefined when setting not found", async () => {
      raw.get.mockRejectedValueOnce(new Error("not found"))
      const result = await store.get("missing")
      expect(result).toBeUndefined()
    })
  })

  describe("set", () => {
    it("resolves on success", async () => {
      raw.post.mockResolvedValueOnce(okResponse({}))
      await expect(store.set("theme", { mode: "light" })).resolves.toBeUndefined()
    })
  })

  describe("delete", () => {
    it("resolves on success", async () => {
      raw.delete.mockResolvedValueOnce(okResponse({}))
      await expect(store.delete("theme")).resolves.toBeUndefined()
    })
  })
})
