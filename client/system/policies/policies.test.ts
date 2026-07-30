import { describe, expect, it, vi, beforeEach } from "vitest"
import { HestiaPolicies } from "./store"
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

describe("HestiaPolicies", () => {
  let policies: HestiaPolicies
  let raw: any

  beforeEach(() => {
    const provider = makeProvider()
    const client = new HttpTransport("http://test.local", "/api")
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
      expect(result.data[0].data?.operationName).toBe("system:apikeys:key:create")
      expect(result.data[0].data?.ruleName).toBe("administrator")
      expect(result.data[0].data?.enabled).toBe(true)
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
      expect(result.data[0].data?.ruleName).toBe("")
      expect(result.data[0].data?.enabled).toBe(false)
    })

    it("maps rateLimit and throttle from doc fields", async () => {
      raw.post.mockResolvedValueOnce(
        okResponse({
          data: [
            {
              _id_: "doc-3",
              _metadata_: { created: "1000", updated: "1000", version: 1, checksum: "abc" },
              operation: "system:auth:session:create",
              rule: "administrator",
              enabled: true,
              protected: true,
              rateLimit: { enabled: true, identity: "ip", capacity: 5, refill: 5, period: 60 },
              throttle: { limit: 10, window: 300, action: { message: "system:users:user:disable", input: { "arguments.id": "{{.claims.user_id}}" } } },
            },
          ],
        }),
      )

      const result = await policies.query({})
      expect(result.data).toHaveLength(1)
      const p = result.data[0].data
      expect(p!.rateLimit).toBeDefined()
      expect(p!.rateLimit!.identity).toBe("ip")
      expect(p!.rateLimit!.capacity).toBe(5)
      expect(p!.throttle).toBeDefined()
      expect(p!.throttle!.limit).toBe(10)
      expect(p!.throttle!.action).toBeDefined()
      expect(p!.throttle!.action!.message).toBe("system:users:user:disable")
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

    it("includes rateLimit and throttle in list response", async () => {
      raw.get.mockResolvedValueOnce(
        okResponse({
          data: {
            policies: [
              {
                id: "p2",
                operationName: "system:auth:session:create",
                ruleName: "administrator",
                enabled: true,
                protected: true,
                rateLimit: { enabled: true, identity: "ip", capacity: 5, refill: 5, period: 60 },
                throttle: { limit: 10, window: 300, action: { message: "system:users:user:disable" } },
              },
            ],
          },
        }),
      )

      const result = await policies.list()
      expect(result.data[0].data?.rateLimit?.identity).toBe("ip")
      expect(result.data[0].data?.throttle?.limit).toBe(10)
    })
  })

  describe("create", () => {
    it("sends rateLimit and throttle in create body", async () => {
      raw.post.mockResolvedValueOnce(
        okResponse({
          data: {
            id: "new-1",
            operationName: "system:auth:session:create",
            ruleName: "administrator",
            enabled: true,
            protected: false,
          },
        }),
      )

      await policies.create({
        data: {
          ruleName: "administrator",
          rateLimit: { enabled: true, identity: "ip", capacity: 5, refill: 5, period: 60 },
          throttle: { limit: 10, window: 300, action: { message: "system:users:user:disable", input: { "arguments.id": "{{.claims.user_id}}" } } },
        },
        options: "system:auth:session:create",
      })

      const callArgs = raw.post.mock.calls[0]
      const body = callArgs[2]?.body ?? callArgs[1]?.body ?? callArgs[1]
      expect(body.rateLimit).toBeDefined()
      expect(body.rateLimit.identity).toBe("ip")
      expect(body.throttle).toBeDefined()
      expect(body.throttle.limit).toBe(10)
    })
  })

  describe("update", () => {
    it("sends rateLimit and throttle in update body", async () => {
      raw.patch.mockResolvedValueOnce(
        okResponse({
          data: {
            id: "upd-1",
            operationName: "system:auth:session:create",
            ruleName: "administrator",
            enabled: true,
            protected: false,
            rateLimit: { enabled: true, identity: "ip", capacity: 10, refill: 10, period: 60 },
          },
        }),
      )

      await policies.update({
        data: {
          rateLimit: { enabled: true, identity: "ip", capacity: 10, refill: 10, period: 60 },
        },
        options: "system:auth:session:create",
      })

      const callArgs = raw.patch.mock.calls[0]
      const body = callArgs[2]?.body ?? callArgs[1]?.body ?? callArgs[1]
      expect(body.rateLimit).toBeDefined()
      expect(body.rateLimit.capacity).toBe(10)
    })
  })
})
