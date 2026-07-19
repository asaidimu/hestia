import { describe, expect, it, vi, beforeEach } from "vitest"
import { HestiaNetworkClient, type IdentityProvider } from "../core/client"
import { HestiaBlobClient, BlobNamespace } from "./store"
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

function errorResponse(status: number): ApiResponse<never> {
  return { success: false, status, data: undefined as never, raw: new Response(null, { status }), headers: new Headers() }
}

function notFoundResponse(): ApiResponse<never> {
  const body = JSON.stringify({ error: { code: "NOT_FOUND", message: "blob not found" } })
  return {
    success: false, status: 404, data: undefined as never,
    raw: new Response(body, { status: 404, headers: { "Content-Type": "application/json" } }),
    headers: new Headers(),
  }
}

describe("BlobNamespace", () => {
  let client: HestiaNetworkClient
  let raw: any
  let ns: BlobNamespace

  beforeEach(() => {
    const provider = makeProvider()
    client = new HestiaNetworkClient("http://test.local", "/api", provider)
    const mock = (createNetworkClient as ReturnType<typeof vi.fn>).mock
    raw = mock.results[mock.results.length - 1]!.value
    vi.clearAllMocks()
    ns = new BlobNamespace(client, "test-bucket")
  })

  describe("upload", () => {
    it("sends POST with blob body and returns document", async () => {
      const file = new File(["hello"], "hello.txt", { type: "text/plain" })
      raw.post.mockResolvedValueOnce(
        okResponse({
          data: { key: "abc", name: "hello.txt", size: 5, content_type: "text/plain", bucket: "test-bucket", created_at: 1000 },
        }),
      )

      const result = await ns.upload({ file, options: { key: "abc" } })
      expect(result!._id_).toBe("abc")
      expect(result!.name).toBe("hello.txt")
      expect(raw.post).toHaveBeenCalledWith(
        "api/system/blobs/blob/test-bucket/abc",
        file,
        { headers: { "Content-Type": "text/plain" }, bodyType: "blob" },
        { type: "blob" },
      )
    })

    it("throws when options.key is missing", async () => {
      await expect(ns.upload({ file: new File([], "x") })).rejects.toThrow("options.key is required")
    })
  })

  describe("read", () => {
    it("fetches metadata by key via head endpoint", async () => {
      raw.post.mockResolvedValueOnce(
        okResponse({
          data: { key: "b1", namespace_id: "test-bucket", content_type: "application/pdf", size: 100, created_at: "2026-01-01T00:00:00Z" },
        }),
      )

      const result = await ns.read("b1")
      expect(result?._id_).toBe("b1")
      expect(result?.namespace_id).toBe("test-bucket")
      expect(raw.post).toHaveBeenCalledWith(
        "api/system/blobs/blob/test-bucket/b1/query",
        undefined,
        { headers: {} },
        undefined,
      )
    })

    it("returns undefined on not found", async () => {
      raw.post.mockResolvedValueOnce(notFoundResponse())

      const result = await ns.read("missing")
      expect(result).toBeUndefined()
    })
  })

  describe("find", () => {
    it("POSTs a query and returns mapped documents", async () => {
      raw.post.mockResolvedValueOnce(
        okResponse({
          data: { blobs: [{ key: "b1", name: "doc.pdf", size: 100, content_type: "application/pdf", bucket: "test-bucket", created_at: 1000 }] },
        }),
      )

      const result = await ns.find()
      expect(result.data).toHaveLength(1)
      expect(result.data[0]._id_).toBe("b1")
      expect(raw.post).toHaveBeenCalledWith(
        "api/system/blobs/blob/test-bucket/query",
        {},
        { headers: {} },
        undefined,
      )
    })
  })

  describe("update", () => {
    it("sends PATCH with key in path and custom data", async () => {
      raw.patch.mockResolvedValueOnce(
        okResponse({
          data: { key: "b1", name: "renamed.pdf", size: 100, content_type: "application/pdf", bucket: "test-bucket", created_at: 1000 },
        }),
      )

      const result = await ns.update({ data: { name: "renamed.pdf" }, options: { key: "b1" } })
      expect(result?.name).toBe("renamed.pdf")
      expect(raw.patch).toHaveBeenCalledWith(
        "api/system/blobs/blob/test-bucket/b1",
        { custom: { name: "renamed.pdf" } },
        { headers: {} },
        undefined,
      )
    })

    it("throws when options.key is missing", async () => {
      await expect(ns.update({ data: { name: "x" } })).rejects.toThrow("options.key is required")
    })
  })

  describe("delete", () => {
    it("sends DELETE with key in path", async () => {
      raw.delete.mockResolvedValueOnce(okResponse({}))
      await ns.delete("b1")
      expect(raw.delete).toHaveBeenCalledWith(
        "api/system/blobs/blob/test-bucket/b1",
        undefined,
        { headers: {} },
        undefined,
      )
    })
  })

  describe("download", () => {
    it("fetches blob with blob responseType", async () => {
      const blob = new Blob(["content"], { type: "application/pdf" })
      raw.get.mockResolvedValueOnce(
        { success: true, status: 200, data: blob, raw: new Response(), headers: new Headers() } as ApiResponse<Blob>,
      )

      const result = await ns.download("b1")
      expect(result.data).toBe(blob)
      expect(result.contentType).toBe("application/pdf")
      expect(raw.get).toHaveBeenCalledWith(
        "api/system/blobs/blob/test-bucket/b1",
        { headers: {}, responseType: "blob" },
      )
    })
  })
})

describe("HestiaBlobClient", () => {
  let blobs: HestiaBlobClient
  let client: HestiaNetworkClient
  let raw: any

  beforeEach(() => {
    const provider = makeProvider()
    client = new HestiaNetworkClient("http://test.local", "/api", provider)
    const mock = (createNetworkClient as ReturnType<typeof vi.fn>).mock
    raw = mock.results[mock.results.length - 1]!.value
    vi.clearAllMocks()
    blobs = new HestiaBlobClient(client, "/api")
  })

  describe("blob (download URL)", () => {
    it("composes download url", () => {
      const url = blobs.blob("test-bucket", "b1")
      expect(url).toBe("http://test.local/api/system/blobs/blob/test-bucket/b1")
    })

    it("composes download url from custom baseUrl and prefix", () => {
      const customClient = new HestiaNetworkClient("http://other.local:9090", "/prefix", makeProvider())
      const customBlobs = new HestiaBlobClient(customClient, "/prefix")
      const url = customBlobs.blob("custom", "x")
      expect(url).toBe("http://other.local:9090/prefix/system/blobs/blob/custom/x")
    })
  })

  describe("namespace", () => {
    it("returns a BlobNamespace instance", () => {
      const ns = blobs.namespace("my-bucket")
      expect(ns).toBeInstanceOf(BlobNamespace)
      expect((ns as any).ns).toBe("my-bucket")
    })
  })
})

describe("HestiaNetworkClient URL composition", () => {
  it("combines baseUrl, prefix and path", async () => {
    const provider = makeProvider()
    const client = new HestiaNetworkClient("http://example.com", "/v2", provider)
    const mock = (createNetworkClient as ReturnType<typeof vi.fn>).mock
    const raw = mock.results[mock.results.length - 1]!.value

    raw.get.mockResolvedValueOnce(okResponse({ data: {} }))
    await client.get("/system/health")
    expect(raw.get).toHaveBeenCalledWith("v2/system/health", { headers: {} })
  })
})

describe("HestiaNetworkClient stream", () => {
  let client: HestiaNetworkClient

  beforeEach(() => {
    const provider = makeProvider()
    client = new HestiaNetworkClient("http://test.local", "/api", provider)
    vi.clearAllMocks()
  })

  it("fires onError for network error", async () => {
    const mockFetch = vi.spyOn(globalThis, "fetch").mockRejectedValue(new Error("net error"))

    const handler = {
      onMessage: vi.fn(),
      onError: vi.fn(),
      onClose: vi.fn(),
    }

    await client.openStream("/test/stream", handler)
    expect(handler.onError).toHaveBeenCalled()
    expect(handler.onClose).not.toHaveBeenCalled()

    mockFetch.mockRestore()
  })

  it("fires onOpen when fetch succeeds", async () => {
    const stream = new ReadableStream({
      start(controller) {
        controller.enqueue(new TextEncoder().encode("data: hello\n\n"))
        controller.close()
      },
    })
    const mockResponse = new Response(stream, { status: 200 })
    const mockFetch = vi.spyOn(globalThis, "fetch").mockResolvedValue(mockResponse)

    const handler = {
      onMessage: vi.fn(),
      onError: vi.fn(),
      onOpen: vi.fn(),
      onClose: vi.fn(),
    }

    await client.openStream("/test/stream", handler)
    expect(handler.onOpen).toHaveBeenCalled()
    expect(handler.onMessage).toHaveBeenCalledWith("hello")
    expect(handler.onClose).toHaveBeenCalled()

    mockFetch.mockRestore()
  })
})
