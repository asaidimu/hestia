import { describe, expect, it, vi, beforeEach } from "vitest"
import { HttpTransport } from "./client"
import { ROUTE_TABLE } from "./routes.gen"

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
import type { ApiResponse } from "@asaidimu/network-client"

function okResponse(data: unknown): ApiResponse<unknown> {
  return { success: true, status: 200, data, raw: new Response(), headers: new Headers() }
}

function fillArgs(spec: { route: string; arguments: readonly string[] }): Record<string, string> {
  const args: Record<string, string> = {}
  for (const a of spec.arguments) args[a] = `arg:${a}`
  return args
}

function expectedPath(spec: { route: string; arguments: readonly string[] }): string {
  const substituted = spec.route.replace(/\{(\w+)\}/g, (_, key: string) => encodeURIComponent(`arg:${key}`))
  return `api${substituted}`
}

const METHOD_TO_RAW = { GET: "get", POST: "post", PATCH: "patch", PUT: "put", DELETE: "delete" } as const

describe("ROUTE_TABLE dispatch coverage", () => {
  let raw: any

  beforeEach(() => {
    const client = new HttpTransport("http://test.local", "/api")
    const mock = (createNetworkClient as ReturnType<typeof vi.fn>).mock
    raw = mock.results[mock.results.length - 1]!.value
    vi.clearAllMocks()
  })

  it("registers every dispatchable command in the route table", () => {
    expect(Object.keys(ROUTE_TABLE).length).toBe(103)
  })

  for (const [name, spec] of Object.entries(ROUTE_TABLE)) {
    it(`dispatch "${name}" uses ${spec.method} on ${spec.route}`, async () => {
      const client = new HttpTransport("http://test.local", "/api")
      const mock = (createNetworkClient as ReturnType<typeof vi.fn>).mock
      raw = mock.results[mock.results.length - 1]!.value
      vi.clearAllMocks()

      const method = METHOD_TO_RAW[spec.method as keyof typeof METHOD_TO_RAW]
      raw[method].mockResolvedValueOnce(okResponse({ data: {} }))

      await client.dispatch(name as any, { arguments: fillArgs(spec) })

      expect(raw[method]).toHaveBeenCalledTimes(1)
      const [path] = raw[method].mock.calls[0] as [string]
      expect(path).toBe(expectedPath(spec))
    })
  }
})
