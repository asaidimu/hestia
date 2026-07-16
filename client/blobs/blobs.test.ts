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
    token: () => null,
    setTokens: vi.fn(),
    setIdentity: vi.fn(),
    clear: vi.fn(),
  }
}

function okResponse<T>(data: T): ApiResponse<T> {
  return { success: true, status: 200, data, raw: new Response() }
}

function errorResponse(status: number): ApiResponse<never> {
  return { success: false, status, data: undefined as never, raw: new Response(null, { status }) }
}

describe("BlobNamespace", () => {
  let client: HestiaNetworkClient
  let raw: ReturnType<typeof createNetworkClient>
  let ns: BlobNamespace

  beforeEach(() => {
    const provider = makeProvider()
    client = new HestiaNetworkClient("http://test.local", provider)
    const mock = (createNetworkClient as ReturnType<typeof vi.fn>).mock
    raw = mock.results[mock.results.length - 1].value
    vi.clearAllMocks()
    ns = new BlobNamespace(client, "test-bucket")
  })

  describe("name", () => {
    it("returns the namespace", () => {
      expect(ns.name()).toBe("test-bucket")
    })
  })

  describe("setPrefix", () => {
    it("sets a prefix filter used by find", async () => {
      ns.setPrefix("images/")
      raw.post.mockResolvedValueOnce(okResponse({ data: { blobs: [] } }))
      await ns.find()
      expect(raw.post).toHaveBeenCalledWith(
        "/system/blobs/blob/test-bucket/query",
        { prefix: "images/" },
        {},
        undefined,
      )
    })
  })

  describe("find", () => {
    it("makes a POST /query and returns a Page of documents", async () => {
      const blobs = [
        { key: "a", namespace_id: "test-bucket", content_type: "text/plain", size: 10, created_at: "2024-01-01T00:00:00Z" },
        { key: "b", namespace_id: "test-bucket", content_type: "application/json", size: 20, created_at: "2024-01-02T00:00:00Z" },
      ]
      raw.post.mockResolvedValueOnce(okResponse({ data: { blobs } }))

      const page = await ns.find()

      expect(raw.post).toHaveBeenCalledWith(
        "/system/blobs/blob/test-bucket/query",
        {},
        {},
        undefined,
      )
      expect(page.data).toHaveLength(2)
      expect(page.data[0]!._id_).toBe("a")
      expect(page.data[0]!.key).toBe("a")
      expect(page.data[1]!._id_).toBe("b")
      expect(page.loading).toBe(false)
      expect(page.page.count).toBe(2)
    })

    it("passes prefix from query DSL", async () => {
      raw.post.mockResolvedValueOnce(okResponse({ data: { blobs: [] } }))
      await ns.find({ prefix: "docs/" } as any)
      expect(raw.post).toHaveBeenCalledWith(
        "/system/blobs/blob/test-bucket/query",
        { prefix: "docs/" },
        {},
        undefined,
      )
    })

    it("passes limit from query DSL", async () => {
      raw.post.mockResolvedValueOnce(okResponse({ data: { blobs: [] } }))
      await ns.find({ pagination: { limit: 10 } } as any)
      expect(raw.post).toHaveBeenCalledWith(
        "/system/blobs/blob/test-bucket/query",
        { limit: 10 },
        {},
        undefined,
      )
    })

    it("returns empty page when response has no blobs", async () => {
      raw.post.mockResolvedValueOnce(okResponse({ data: { blobs: undefined } }))
      const page = await ns.find()
      expect(page.data).toEqual([])
    })
  })

  describe("head", () => {
    const meta = {
      key: "myfile",
      namespace_id: "test-bucket",
      content_type: "image/png",
      size: 1024,
      created_at: "2024-01-01T00:00:00Z",
    }

    it("makes a GET and returns a BlobDocument", async () => {
      raw.get.mockResolvedValueOnce(okResponse({ data: meta }))

      const doc = await ns.head("myfile")

      expect(raw.get).toHaveBeenCalledWith(
        "/system/blobs/blob/test-bucket/myfile",
        {},
      )
      expect(doc).toBeDefined()
      expect(doc!._id_).toBe("myfile")
      expect(doc!.content_type).toBe("image/png")
    })

    it("returns undefined on 404", async () => {
      raw.get.mockResolvedValueOnce(
        okResponse({ data: undefined }),
      )

      const doc = await ns.head("missing")
      expect(doc).toBeUndefined()
    })

    it("returns undefined on not-found error code", async () => {
      raw.get.mockRejectedValueOnce({
        code: "SYNC-001-NF",
        message: "not found",
      })

      const doc = await ns.head("missing")
      expect(doc).toBeUndefined()
    })

    it("returns undefined on INTERNAL_ERROR with not-found message", async () => {
      raw.get.mockRejectedValueOnce({
        code: "INTERNAL_ERROR",
        message: "blob not found",
      })

      const doc = await ns.head("missing")
      expect(doc).toBeUndefined()
    })

    it("rethrows other errors", async () => {
      raw.get.mockRejectedValueOnce(new Error("network error"))

      await expect(ns.head("myfile")).rejects.toThrow("network error")
    })
  })

  describe("upload", () => {
    it("makes a POST with blob bodyType and returns a BlobDocument", async () => {
      const meta = {
        key: "photo.jpg",
        namespace_id: "test-bucket",
        content_type: "image/jpeg",
        size: 5000,
        created_at: "2024-01-01T00:00:00Z",
      }
      raw.post.mockResolvedValueOnce(okResponse({ data: meta }))

      const blob = new Blob(["fake-image-data"], { type: "image/jpeg" })
      const doc = await ns.upload("photo.jpg", blob, "image/jpeg")

      expect(raw.post).toHaveBeenCalledWith(
        "/system/blobs/blob/test-bucket/photo.jpg",
        blob,
        { headers: { "Content-Type": "image/jpeg" }, bodyType: "blob" },
        { type: "blob" },
      )
      expect(doc._id_).toBe("photo.jpg")
    })

    it("uses blob.type when contentType not provided", async () => {
      raw.post.mockResolvedValueOnce(okResponse({ data: { key: "f", namespace_id: "test-bucket", content_type: "text/plain", size: 3, created_at: "2024-01-01T00:00:00Z" } }))

      const blob = new Blob(["abc"], { type: "text/plain" })
      await ns.upload("f", blob)

      expect(raw.post).toHaveBeenCalledWith(
        "/system/blobs/blob/test-bucket/f",
        blob,
        { headers: { "Content-Type": "text/plain" }, bodyType: "blob" },
        { type: "blob" },
      )
    })

    it("omits Content-Type when neither contentType nor blob.type is set", async () => {
      raw.post.mockResolvedValueOnce(okResponse({ data: { key: "f", namespace_id: "test-bucket", content_type: "", size: 3, created_at: "2024-01-01T00:00:00Z" } }))

      const blob = new Blob(["abc"])
      await ns.upload("f", blob)

      expect(raw.post).toHaveBeenCalledWith(
        "/system/blobs/blob/test-bucket/f",
        blob,
        { headers: {}, bodyType: "blob" },
        { type: "blob" },
      )
    })
  })

  describe("download", () => {
    it("makes a GET with blob responseType and returns blob + contentType", async () => {
      const blob = new Blob(["file-content"], { type: "application/pdf" })
      raw.get.mockResolvedValueOnce(okResponse(blob))

      const result = await ns.download("report.pdf")

      expect(raw.get).toHaveBeenCalledWith(
        "/system/blobs/blob/test-bucket/report.pdf",
        { responseType: "blob" },
      )
      expect(result.data).toBe(blob)
      expect(result.contentType).toBe("application/pdf")
    })
  })

  describe("updateMetadata", () => {
    it("makes a PATCH with custom metadata", async () => {
      const meta = {
        key: "doc",
        namespace_id: "test-bucket",
        content_type: "text/plain",
        size: 100,
        created_at: "2024-01-01T00:00:00Z",
        custom: { author: "alice", project: "x" },
      }
      raw.patch.mockResolvedValueOnce(okResponse({ data: meta }))

      const result = await ns.updateMetadata("doc", { author: "alice", project: "x" })

      expect(raw.patch).toHaveBeenCalledWith(
        "/system/blobs/blob/test-bucket/doc",
        { custom: { author: "alice", project: "x" } },
        {},
        undefined,
      )
      expect(result.custom).toEqual({ author: "alice", project: "x" })
    })
  })

  describe("delete", () => {
    it("makes a DELETE request", async () => {
      raw.delete.mockResolvedValueOnce(okResponse(null))

      await ns.delete("obsolete.txt")

      expect(raw.delete).toHaveBeenCalledWith(
        "/system/blobs/blob/test-bucket/obsolete.txt",
        undefined,
        {},
        undefined,
      )
    })
  })

  describe("list", () => {
    it("makes a POST /query and returns raw BlobMeta array", async () => {
      const blobs = [
        { key: "a", namespace_id: "test-bucket", content_type: "text/plain", size: 1, created_at: "2024-01-01T00:00:00Z" },
      ]
      raw.post.mockResolvedValueOnce(okResponse({ data: { blobs } }))

      const result = await ns.list()

      expect(raw.post).toHaveBeenCalledWith(
        "/system/blobs/blob/test-bucket/query",
        {},
        {},
        undefined,
      )
      expect(result).toHaveLength(1)
      expect(result[0]!.key).toBe("a")
    })

    it("returns empty array when no blobs key in response", async () => {
      raw.post.mockResolvedValueOnce(okResponse({ data: {} }))

      const result = await ns.list()
      expect(result).toEqual([])
    })
  })

  describe("page", () => {
    it("returns a PagedData controller", () => {
      const pager = ns.page()
      expect(pager).toBeDefined()
      expect(typeof pager.page).toBe("function")
      expect(typeof pager.navigate).toBe("function")
      expect(typeof pager.refresh).toBe("function")
    })
  })
})

describe("HestiaBlobClient", () => {
  let client: HestiaNetworkClient
  let raw: ReturnType<typeof createNetworkClient>
  let blobClient: HestiaBlobClient

  beforeEach(() => {
    const provider = makeProvider()
    client = new HestiaNetworkClient("http://test.local", provider)
    const mock = (createNetworkClient as ReturnType<typeof vi.fn>).mock
    raw = mock.results[mock.results.length - 1].value
    vi.clearAllMocks()
    blobClient = new HestiaBlobClient(client)
  })

  describe("namespaces", () => {
    it("makes a POST /namespace/query and returns namespace list", async () => {
      const namespaces = [
        { id: "ns1", display_name: "Bucket 1" },
        { id: "ns2", display_name: "Bucket 2" },
      ]
      raw.post.mockResolvedValueOnce(okResponse({ data: { namespaces } }))

      const result = await blobClient.namespaces()

      expect(raw.post).toHaveBeenCalledWith(
        "/system/blobs/namespace/query",
        undefined,
        {},
        undefined,
      )
      expect(result).toHaveLength(2)
      expect(result[0]!.id).toBe("ns1")
    })

    it("returns empty array when no namespaces key", async () => {
      raw.post.mockResolvedValueOnce(okResponse({ data: {} }))

      const result = await blobClient.namespaces()
      expect(result).toEqual([])
    })
  })

  describe("createNamespace", () => {
    it("makes a POST and returns the created namespace", async () => {
      const nsInfo = { id: "new-ns", display_name: "New Bucket" }
      raw.post.mockResolvedValueOnce(okResponse({ data: nsInfo }))

      const result = await blobClient.createNamespace({ display_name: "New Bucket" })

      expect(raw.post).toHaveBeenCalledWith(
        "/system/blobs/namespace",
        { display_name: "New Bucket" },
        {},
        undefined,
      )
      expect(result.id).toBe("new-ns")
    })
  })

  describe("deleteNamespace", () => {
    it("makes a DELETE request", async () => {
      raw.delete.mockResolvedValueOnce(okResponse(null))

      await blobClient.deleteNamespace("old-ns")

      expect(raw.delete).toHaveBeenCalledWith(
        "/system/blobs/namespace/old-ns",
        undefined,
        {},
        undefined,
      )
    })

    it("encodes special characters in namespace name", async () => {
      raw.delete.mockResolvedValueOnce(okResponse(null))

      await blobClient.deleteNamespace("my namespace")

      expect(raw.delete).toHaveBeenCalledWith(
        "/system/blobs/namespace/my%20namespace",
        undefined,
        {},
        undefined,
      )
    })
  })

  describe("blob", () => {
    it("returns the full URL for a blob (key is URI-encoded)", () => {
      const url = blobClient.blob("my-bucket", "path/to/file.txt")
      expect(url).toBe(
        "http://test.local/system/blobs/blob/my-bucket/path%2Fto%2Ffile.txt",
      )
    })
  })

  describe("namespace", () => {
    it("returns a BlobNamespace instance for the given namespace", () => {
      const ns = blobClient.namespace("my-bucket")
      expect(ns).toBeInstanceOf(BlobNamespace)
      expect(ns.name()).toBe("my-bucket")
    })
  })
})
