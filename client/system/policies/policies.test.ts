import { describe, expect, it, vi, beforeEach } from "vitest"
import { HestiaPolicies } from "./store"
import { HestiaNetworkClient, type IdentityProvider } from "../../core/client"
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

describe("HestiaPolicies", () => {
  let policies: HestiaPolicies
  let raw: any

  beforeEach(() => {
    const provider = makeProvider()
    const client = new HestiaNetworkClient("http://test.local", "/api", provider)
    const mock = (createNetworkClient as ReturnType<typeof vi.fn>).mock
    raw = mock.results[mock.results.length - 1]!.value
    vi.clearAllMocks()
    policies = new HestiaPolicies(client)
  })

  describe("setEnabled", () => {
    it("sends PATCH with enabled:false and preserves all fields in response", async () => {
      raw.patch.mockResolvedValueOnce(
        okResponse({
          data: {
            id: "019f7b30bc9b7ec4bc0d245db4b1e0f0",
            operationName: "system:apikeys:key:create",
            ruleName: "administrator",
            enabled: false,
            protected: true,
          },
        }),
      )

      const result = await policies.setEnabled("system:apikeys:key:create", false)

      expect(raw.patch).toHaveBeenCalledWith(
        "api/system/policies/policy/system%3Aapikeys%3Akey%3Acreate",
        { enabled: false },
        { headers: {} },
        undefined,
      )

      expect(result.data.enabled).toBe(false)
      expect(result.data.ruleName).toBe("administrator")
      expect(result.data.operationName).toBe("system:apikeys:key:create")
      expect(result.data.id).toBe("019f7b30bc9b7ec4bc0d245db4b1e0f0")
      expect(result.data.protected).toBe(true)
    })

    it("sends PATCH with enabled:true", async () => {
      raw.patch.mockResolvedValueOnce(
        okResponse({
          data: {
            id: "019f7b30bc9b7ec4bc0d245db4b1e0f0",
            operationName: "system:apikeys:key:create",
            ruleName: "administrator",
            enabled: true,
            protected: true,
          },
        }),
      )

      const result = await policies.setEnabled("system:apikeys:key:create", true)

      expect(raw.patch).toHaveBeenCalledWith(
        "api/system/policies/policy/system%3Aapikeys%3Akey%3Acreate",
        { enabled: true },
        { headers: {} },
        undefined,
      )
      expect(result.data.enabled).toBe(true)
    })
  })

  describe("query", () => {
    it("maps raw doc fields (operation, rule) to Policy fields", async () => {
      raw.post.mockResolvedValueOnce(
        okResponse({
          data: [
            {
              _id_: "doc-1",
              _metadata_: { created: "1000", updated: "1000", version: 1, checksum: "abc" },
              operation: "system:apikeys:key:create",
              rule: "administrator",
              enabled: true,
              protected: true,
              description: null,
              intentType: null,
            },
          ],
        }),
      )

      const result = await policies.query({})
      expect(result.data).toHaveLength(1)
      expect(result.data[0].operationName).toBe("system:apikeys:key:create")
      expect(result.data[0].ruleName).toBe("administrator")
      expect(result.data[0].enabled).toBe(true)
    })

    it("defaults ruleName to empty string when rule field is missing", async () => {
      raw.post.mockResolvedValueOnce(
        okResponse({
          data: [
            {
              _id_: "doc-2",
              _metadata_: { created: "1000", updated: "1000", version: 1, checksum: "abc" },
              operation: "system:test:op",
              enabled: false,
              protected: false,
            },
          ],
        }),
      )

      const result = await policies.query({})
      expect(result.data[0].ruleName).toBe("")
      expect(result.data[0].enabled).toBe(false)
    })
  })

  describe("list", () => {
    it("returns policies from server response", async () => {
      raw.get.mockResolvedValueOnce(
        okResponse({
          data: {
            policies: [
              { id: "p1", operationName: "system:test:op", ruleName: "administrator", enabled: true, protected: true },
            ],
          },
        }),
      )

      const result = await policies.list()
      expect(result.data).toHaveLength(1)
      expect(result.data[0].data?.operationName).toBe("system:test:op")
    })
  })
})
