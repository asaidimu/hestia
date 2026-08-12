import { describe, expect, it, vi, beforeEach } from "vitest"
import { HttpTransport, type IdentityProvider } from "../core/client"
import {
  AimdController,
  BlobNamespace,
  getMissingRanges,
  HestiaBlobClient,
  ResumableUpload,
  UploadJobStore,
  UploadQueue,
  type UploadTransport,
} from "./store"
import type { Document } from "../core/types"
import type { SimplePersistence } from "@asaidimu/utils-persistence"
import type { ApiResponse } from "@asaidimu/network-client"
import type {
  BlobMeta,
  UploadBeginResult,
  UploadJobSnapshot,
  UploadPersistence,
  UploadProgressResult,
} from "./types"

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

/**
 * Stateful staged-upload mock. Tracks received chunk offsets so the server's
 * `progress` response reflects the real accumulated state, matching the
 * engine's loop (progress → pool → progress → …) instead of a fixed queue.
 */
function mockStagedUpload(raw: any, blockSize: number, fileSize: number) {
  const received: { offset: number; size: number }[] = []
  const chunkCalls: { offset: number; size: number }[] = []
  let begin = false

  const receivedTotal = () => {
    const merged: { start: number; end: number }[] = []
    for (const c of [...received].sort((a, b) => a.offset - b.offset)) {
      const r = { start: c.offset, end: c.offset + c.size }
      const last = merged[merged.length - 1]
      if (last && r.start <= last.end) last.end = Math.max(last.end, r.end)
      else merged.push(r)
    }
    return {
      total: merged.reduce((s, r) => s + (r.end - r.start), 0),
      ranges: merged,
      block_size: blockSize,
      expected_size: fileSize,
    }
  }

  raw.post.mockImplementation(async (path: string, _body: unknown, opts: any) => {
    if (path.includes("blob/begin")) {
      begin = true
      return okResponse({ data: { session_id: "s1", key: "abc", offset: 0, block_size: blockSize } })
    }
    if (path.includes("blob/progress")) {
      if (!begin) throw new Error("progress before begin")
      return okResponse({ data: receivedTotal() })
    }
    if (path.includes("blob/chunk")) {
      const offset = Number(opts.headers["X-Offset"])
      const size = Math.min(blockSize, fileSize - offset)
      received.push({ offset, size })
      chunkCalls.push({ offset, size })
      return okResponse({ data: { total: receivedTotal().total } })
    }
    if (path.includes("blob/complete")) {
      return okResponse({
        data: {
          key: "abc",
          namespace_id: "test-bucket",
          content_type: "application/octet-stream",
          size: fileSize,
          created_at: "2026-01-01T00:00:00Z",
        },
      })
    }
    throw new Error(`unexpected staged call: ${path}`)
  })

  return {
    received,
    chunkOffsets: () => chunkCalls.map((c) => c.offset),
    totalBytes: () => receivedTotal().total,
  }
}

function fakeDoc(size: number): Document<BlobMeta> {
  return {
    _id_: "abc",
    _metadata_: { checksum: "", created: "2026-01-01T00:00:00Z", updated: "2026-01-01T00:00:00Z", version: 1 },
    key: "abc",
    namespace_id: "ns",
    content_type: "application/octet-stream",
    size,
    created_at: "2026-01-01T00:00:00Z",
  }
}

/**
 * In-memory transport that tracks received ranges per session. Chunk calls can
 * be gated (block until released), hung (only released by abort), or failed on
 * demand to exercise retry and state-machine paths deterministically.
 */
class FakeTransport implements UploadTransport {
  receivedBySession = new Map<string, { offset: number; size: number }[]>()
  sessionCounter = 0
  beginCalls = 0
  completeCalls = 0
  abortCalls = 0
  chunkGate?: Promise<void>
  hangFirstChunk = false
  failChunkOnce = false
  failCompleteOnce = false

  constructor(
    public fileSize: number,
    public blockSize = 8,
  ) {}

  private session(sessionId: string): { offset: number; size: number }[] {
    let list = this.receivedBySession.get(sessionId)
    if (!list) {
      list = []
      this.receivedBySession.set(sessionId, list)
    }
    return list
  }

  private merged(list: { offset: number; size: number }[]): { start: number; end: number }[] {
    const out: { start: number; end: number }[] = []
    for (const c of [...list].sort((a, b) => a.offset - b.offset)) {
      const r = { start: c.offset, end: c.offset + c.size }
      const last = out[out.length - 1]
      if (last && r.start <= last.end) last.end = Math.max(last.end, r.end)
      else out.push(r)
    }
    return out
  }

  private totalFor(sessionId: string): number {
    return this.merged(this.session(sessionId)).reduce((s, r) => s + (r.end - r.start), 0)
  }

  async beginUpload(props: {
    key: string
    size: number
    contentType?: string
    blockSize?: number
    overwrite?: boolean
  }): Promise<UploadBeginResult> {
    this.beginCalls++
    this.sessionCounter++
    return {
      session_id: `s${this.sessionCounter}`,
      key: props.key,
      offset: 0,
      block_size: this.blockSize,
    }
  }

  async uploadChunk(props: {
    sessionId: string
    offset: number
    data: Blob
    sha256?: string
    signal?: AbortSignal
  }): Promise<number> {
    if (this.hangFirstChunk) {
      this.hangFirstChunk = false
      return new Promise<number>((_, reject) => {
        if (props.signal?.aborted) {
          reject(new DOMException("Aborted", "AbortError"))
          return
        }
        props.signal?.addEventListener(
          "abort",
          () => reject(new DOMException("Aborted", "AbortError")),
          { once: true },
        )
      })
    }
    if (this.chunkGate) await this.chunkGate
    if (this.failChunkOnce) {
      this.failChunkOnce = false
      throw new Error("chunk boom")
    }
    const list = this.session(props.sessionId)
    list.push({ offset: props.offset, size: props.data.size })
    return this.totalFor(props.sessionId)
  }

  async completeUpload(props: { sessionId: string; overwrite?: boolean }): Promise<Document<BlobMeta> | undefined> {
    this.completeCalls++
    if (this.failCompleteOnce) {
      this.failCompleteOnce = false
      throw new Error("complete boom")
    }
    return fakeDoc(this.fileSize)
  }

  async progress(sessionId: string): Promise<UploadProgressResult> {
    const list = this.session(sessionId)
    return {
      total: this.totalFor(sessionId),
      ranges: this.merged(list),
      block_size: this.blockSize,
      expected_size: this.fileSize,
    }
  }

  async abort(_sessionId: string): Promise<void> {
    this.abortCalls++
  }
}

function memPersistence<T>(): SimplePersistence<T> {
  let state: T | null = null
  return {
    set: vi.fn(async (_id: string, s: T) => {
      state = s
      return true
    }),
    get: vi.fn(async () => state),
    subscribe: () => () => {},
    clear: vi.fn(async () => {
      state = null
      return true
    }),
    stats: () => ({ version: "1.0.0", id: "mem" }),
  }
}

function deferred<T = void>(): { promise: Promise<T>; resolve: (v: T) => void } {
  let resolve!: (v: T) => void
  const promise = new Promise<T>((r) => {
    resolve = r
  })
  return { promise, resolve }
}

async function sha256HexTest(data: ArrayBuffer | Uint8Array): Promise<string> {
  const digest = await crypto.subtle.digest("SHA-256", data as any)
  return Array.from(new Uint8Array(digest))
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("")
}

describe("BlobNamespace", () => {
  let client: HttpTransport
  let raw: any
  let ns: BlobNamespace

  beforeEach(() => {
    const provider = makeProvider()
    client = new HttpTransport("http://test.local", "/api")
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
      expect(result!.content_type).toBe("text/plain")
    })

    it("throws when options.key is missing", async () => {
      await expect(ns.upload({ file: new File([], "x") })).rejects.toThrow("options.key is required")
    })
  })

  describe("resumable upload protocol", () => {
    it("beginUpload sends begin with key/size/content_type/block_size", async () => {
      raw.post.mockResolvedValueOnce(
        okResponse({ data: { session_id: "s1", key: "abc", offset: 0, block_size: 8 } }),
      )
      const r = await ns.beginUpload({ key: "abc", size: 20, contentType: "text/plain", blockSize: 8 })
      expect(r.session_id).toBe("s1")
      const [path, body] = raw.post.mock.calls[0] as [string, any]
      expect(path).toBe("api/system/blobs/blob/begin/test-bucket")
      expect(body).toMatchObject({ key: "abc", size: 20, content_type: "text/plain", block_size: 8 })
    })

    it("beginUpload appends overwrite modifier", async () => {
      raw.post.mockResolvedValueOnce(
        okResponse({ data: { session_id: "s1", key: "abc", offset: 0, block_size: 8 } }),
      )
      await ns.beginUpload({ key: "abc", size: 1, overwrite: true })
      expect(raw.post.mock.calls[0]![0]).toContain("overwrite=true")
    })

    it("uploadChunk sends headers and blob body", async () => {
      raw.post.mockResolvedValueOnce(okResponse({ data: { total: 8 } }))
      const data = new Blob([new Uint8Array(8)])
      const total = await ns.uploadChunk({ sessionId: "s1", offset: 0, data, sha256: "deadbeef" })
      expect(total).toBe(8)
      const [path, body, opts, bodyOpts] = raw.post.mock.calls[0] as [string, unknown, any, any]
      expect(path).toBe("api/system/blobs/blob/chunk/test-bucket")
      expect(opts.headers).toMatchObject({ "X-Session-ID": "s1", "X-Offset": "0", "X-Chunk-SHA256": "deadbeef" })
      expect(bodyOpts).toEqual({ type: "blob" })
      expect(body).toBe(data)
    })

    it("completeUpload sends session header", async () => {
      raw.post.mockResolvedValueOnce(
        okResponse({
          data: { key: "abc", namespace_id: "test-bucket", content_type: "text/plain", size: 20, created_at: "2026-01-01T00:00:00Z" },
        }),
      )
      const doc = await ns.completeUpload({ sessionId: "s1" })
      expect(doc?._id_).toBe("abc")
      const [path, , opts] = raw.post.mock.calls[0] as [string, unknown, any]
      expect(path).toBe("api/system/blobs/blob/complete/test-bucket")
      expect(opts.headers).toMatchObject({ "X-Session-ID": "s1" })
    })

    it("progress reports received ranges", async () => {
      raw.post.mockResolvedValueOnce(
        okResponse({ data: { total: 8, ranges: [{ start: 0, end: 8 }], block_size: 8, expected_size: 20 } }),
      )
      const p = await ns.progress("s1")
      expect(p.total).toBe(8)
      expect(p.ranges).toEqual([{ start: 0, end: 8 }])
    })

    it("abort sends session header", async () => {
      raw.post.mockResolvedValueOnce(okResponse({}))
      await ns.abort("s1")
      const [path, , opts] = raw.post.mock.calls[0] as [string, unknown, any]
      expect(path).toBe("api/system/blobs/blob/abort/test-bucket")
      expect(opts.headers).toMatchObject({ "X-Session-ID": "s1" })
    })
  })

  describe("resumable upload flow", () => {
    it("uploads missing blocks with offsets and sha256, then completes", async () => {
      const file = new File([new Uint8Array(20)], "big.bin", { type: "application/octet-stream" })
      const staged = mockStagedUpload(raw, 8, 20)

      const doc = await ns.upload({ file, options: { key: "abc", forceStaged: true, blockSize: 8 } })
      expect(doc?._id_).toBe("abc")

      expect(staged.chunkOffsets()).toEqual([0, 8, 16])
      const chunkCalls = (raw.post.mock.calls as [string, unknown, any][])
        .filter(([path]) => path.includes("blob/chunk"))
      expect(chunkCalls).toHaveLength(3)
      for (const [, , opts] of chunkCalls) {
        expect(opts.headers["X-Session-ID"]).toBe("s1")
        expect(opts.headers["X-Chunk-SHA256"]).toMatch(/^[0-9a-f]{64}$/)
      }
    })

    it("skips blocks already received on resume", async () => {
      const file = new File([new Uint8Array(20)], "big.bin")
      const staged = mockStagedUpload(raw, 8, 20)
      staged.received.push({ offset: 0, size: 8 })

      await ns.upload({ file, options: { key: "abc", forceStaged: true, blockSize: 8 } })

      expect(staged.chunkOffsets()).toEqual([8, 16])
    })

    it("fires onProgress with cumulative bytes", async () => {
      const file = new File([new Uint8Array(20)], "big.bin")
      mockStagedUpload(raw, 8, 20)

      const onProgress = vi.fn()
      await ns.upload({ file, options: { key: "abc", forceStaged: true, blockSize: 8, onProgress } })
      expect(onProgress).toHaveBeenCalled()
      const last = onProgress.mock.calls[onProgress.mock.calls.length - 1]![0]
      expect(last.uploaded).toBe(20)
      expect(last.total).toBe(20)
    })

    it("uses direct upload for small files below threshold", async () => {
      const file = new File(["hello"], "hello.txt", { type: "text/plain" })
      raw.post.mockResolvedValueOnce(
        okResponse({
          data: { key: "abc", namespace_id: "test-bucket", content_type: "text/plain", size: 5, created_at: "2026-01-01T00:00:00Z" },
        }),
      )
      await ns.upload({ file, options: { key: "abc" } })
      expect(raw.post.mock.calls[0]![0]).toContain("blob/upload")
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
      expect(result.data[0]!._id_).toBe("b1")
    })
  })

  describe("update", () => {
    it("sends PATCH with key in path and custom data", async () => {
      raw.patch.mockResolvedValueOnce(
        okResponse({
          data: { key: "b1", namespace_id: "test-bucket", content_type: "application/pdf", size: 100, created_at: "2026-01-01T00:00:00Z" },
        }),
      )

      const result = await ns.update({ data: { content_type: "application/pdf" }, options: { key: "b1" } })
      expect(result?.content_type).toBe("application/pdf")
    })

    it("throws when options.key is missing", async () => {
      await expect(ns.update({ data: { content_type: "text/plain" } })).rejects.toThrow("options.key is required")
    })
  })

  describe("delete", () => {
    it("sends DELETE with key in path", async () => {
      raw.delete.mockResolvedValueOnce(okResponse({}))

      await ns.delete("b1")
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
    })
  })
})

describe("getMissingRanges", () => {
  it("fills gaps between received ranges", () => {
    expect(
      getMissingRanges(
        [
          { start: 0, end: 8 },
          { start: 16, end: 20 },
        ],
        24,
      ),
    ).toEqual([
      { start: 8, end: 16 },
      { start: 20, end: 24 },
    ])
  })

  it("returns the whole file when nothing is received", () => {
    expect(getMissingRanges([], 20)).toEqual([{ start: 0, end: 20 }])
  })

  it("returns nothing when the file is fully received", () => {
    expect(getMissingRanges([{ start: 0, end: 20 }], 20)).toEqual([])
  })
})

describe("AimdController", () => {
  it("grows concurrency after a success streak", () => {
    const c = new AimdController(4)
    c.recordSuccess()
    c.recordSuccess()
    c.recordSuccess()
    expect(c.maxConcurrency).toBe(5)
  })

  it("caps concurrency at the maximum", () => {
    const c = new AimdController(6)
    for (let i = 0; i < 12; i++) c.recordSuccess()
    expect(c.maxConcurrency).toBe(6)
  })

  it("halves concurrency on failure", () => {
    const c = new AimdController(6)
    c.recordFailure()
    expect(c.maxConcurrency).toBe(3)
  })

  it("does not drop below the minimum concurrency", () => {
    const c = new AimdController(1)
    c.recordFailure()
    expect(c.maxConcurrency).toBe(1)
  })

  it("gates rapid consecutive failures behind a cooldown", () => {
    const c = new AimdController(6)
    c.recordFailure()
    c.recordFailure()
    c.recordFailure()
    expect(c.maxConcurrency).toBe(3)
  })

  it("tracks an EWMA throughput estimate", () => {
    const c = new AimdController(4)
    c.recordThroughput(1024 * 1024)
    expect(c.estThroughput()).toBeGreaterThan(256 * 1024)
    c.recordThroughput(32 * 1024)
    expect(c.estThroughput()).toBeLessThan(1024 * 1024)
  })
})

function testFile(): File {
  return new File([new Uint8Array(20)], "big.bin")
}

describe("ResumableUpload engine", () => {
  it("streams pre-hash and reports hash->upload phase transition with correct fingerprint", async () => {
    const bytes = new Uint8Array(20).fill(0xab)
    const f = new File([bytes], "big.bin")
    const t = new FakeTransport(20, 8)
    const phases: string[] = []
    const progress: any[] = []

    const upload = new ResumableUpload(t, f, "abc", {
      key: "abc",
      forceStaged: true,
      blockSize: 8,
      concurrency: 1,
      onProgress: (p) => {
        phases.push(p.phase)
        progress.push(p)
      },
    })
    upload.start()

    const doc = await upload.done
    expect(doc?._id_).toBe("abc")

    expect(phases).toContain("hash")
    expect(phases).toContain("upload")
    const hashPhase = progress.find((p) => p.phase === "hash")
    expect(hashPhase).toBeDefined()
    expect(progress.filter((p) => p.phase === "hash").some((p) => p.hashed === 20)).toBe(true)
    expect(hashPhase.hashTotal).toBe(20)
    expect(hashPhase.percent).toBeLessThanOrEqual(1)

    const blockHashes: string[] = []
    for (let i = 0; i < 3; i++) {
      const start = i * 8
      const end = Math.min(start + 8, 20)
      const buf = await new Blob([bytes.slice(start, end)]).arrayBuffer()
      blockHashes.push(await sha256HexTest(buf))
    }
    const expected = await sha256HexTest(new TextEncoder().encode(blockHashes.join("")))
    const withFingerprint = progress.find((p) => p.fingerprint)
    expect(withFingerprint?.fingerprint).toBe(expected)
    expect(withFingerprint?.fingerprint).toMatch(/^[0-9a-f]{64}$/)
  })

  it("uploads all missing blocks with offsets and sha256 then completes", async () => {
    const t = new FakeTransport(20, 8)
    const upload = new ResumableUpload(t, testFile(), "abc", {
      key: "abc",
      forceStaged: true,
      blockSize: 8,
      concurrency: 1,
    })
    upload.start()
    const doc = await upload.done

    expect(doc?._id_).toBe("abc")
    expect(upload.status).toBe("completed")
    expect(t.beginCalls).toBe(1)
    expect(t.completeCalls).toBe(1)
    expect(t.receivedBySession.get("s1")!.map((c) => c.offset)).toEqual([0, 8, 16])
    expect(upload.uploaded).toBe(20)
  })

  it("skips blocks already received on resume", async () => {
    const t = new FakeTransport(20, 8)
    t.receivedBySession.set("s1", [{ offset: 0, size: 8 }])

    const upload = new ResumableUpload(t, testFile(), "abc", {
      key: "abc",
      forceStaged: true,
      blockSize: 8,
      concurrency: 1,
    })
    upload.start()
    await upload.done

    expect(t.receivedBySession.get("s1")!.map((c) => c.offset)).toEqual([0, 8, 16])
    expect(upload.uploaded).toBe(20)
  })

  it("retries a failed chunk and recovers", async () => {
    const t = new FakeTransport(20, 8)
    t.failChunkOnce = true
    const upload = new ResumableUpload(t, testFile(), "abc", {
      key: "abc",
      forceStaged: true,
      blockSize: 8,
      concurrency: 1,
      retries: 2,
    })
    upload.start()
    const doc = await upload.done

    expect(doc?._id_).toBe("abc")
    expect(upload.status).toBe("completed")
    expect(t.receivedBySession.get("s1")!.map((c) => c.offset)).toEqual([0, 8, 16])
  }, 8000)

  it("retries a failed completion", async () => {
    const t = new FakeTransport(20, 8)
    t.failCompleteOnce = true
    const upload = new ResumableUpload(t, testFile(), "abc", {
      key: "abc",
      forceStaged: true,
      blockSize: 8,
      concurrency: 1,
    })
    upload.start()
    const err = await upload.done.catch((e) => e)
    expect(err).toBeInstanceOf(Error)
    expect(upload.status).toBe("error")

    upload.retry()
    await vi.waitFor(() => expect(upload.status).toBe("completed"))
    expect(t.completeCalls).toBe(2)
  })

  it("done reflects a retried attempt after an error", async () => {
    const t = new FakeTransport(20, 8)
    t.failCompleteOnce = true
    const upload = new ResumableUpload(t, testFile(), "abc", {
      key: "abc",
      forceStaged: true,
      blockSize: 8,
      concurrency: 1,
    })
    upload.start()
    const first = await upload.done.catch((e) => e)
    expect(first).toBeInstanceOf(Error)
    expect(upload.status).toBe("error")

    upload.retry()
    const doc = await upload.done
    expect(doc?._id_).toBe("abc")
    expect(upload.status).toBe("completed")
  })

  it("pauses mid-upload then resumes to completion", async () => {
    const t = new FakeTransport(20, 8)
    t.hangFirstChunk = true
    const upload = new ResumableUpload(t, testFile(), "abc", {
      key: "abc",
      forceStaged: true,
      blockSize: 8,
      concurrency: 1,
    })
    upload.start()
    await vi.waitFor(() => expect(upload.status).toBe("uploading"))

    upload.pause()
    await vi.waitFor(() => expect(upload.status).toBe("paused"))
    expect(upload.sessionId).toBe("s1")

    upload.resume()
    const doc = await upload.done
    expect(doc?._id_).toBe("abc")
    expect(upload.status).toBe("completed")
  })

  it("cancels mid-upload and aborts the server session", async () => {
    const t = new FakeTransport(20, 8)
    t.hangFirstChunk = true
    const upload = new ResumableUpload(t, testFile(), "abc", {
      key: "abc",
      forceStaged: true,
      blockSize: 8,
      concurrency: 1,
    })
    upload.start()
    await vi.waitFor(() => expect(upload.status).toBe("uploading"))

    await upload.cancel()
    await vi.waitFor(() => expect(upload.status).toBe("cancelled"))
    expect(t.abortCalls).toBe(1)

    const doc = await upload.done
    expect(doc).toBeUndefined()
  })

  it("cancels immediately when constructed with an already-aborted signal", async () => {
    const t = new FakeTransport(20, 8)
    const ac = new AbortController()
    ac.abort()
    const upload = new ResumableUpload(t, testFile(), "abc", {
      key: "abc",
      forceStaged: true,
      signal: ac.signal,
    })
    const doc = await upload.done
    expect(doc).toBeUndefined()
    expect(upload.status).toBe("cancelled")
  })
})

describe("UploadQueue engine", () => {
  it("runs at most maxConcurrent uploads at once", async () => {
    const t = new FakeTransport(8, 8)
    const gate = deferred()
    t.chunkGate = gate.promise

    const q = new UploadQueue(t, "ns", { maxConcurrent: 2 })
    const u1 = q.enqueue(new File([new Uint8Array(8)], "a.bin"), { key: "a", forceStaged: true, blockSize: 8 })
    const u2 = q.enqueue(new File([new Uint8Array(8)], "b.bin"), { key: "b", forceStaged: true, blockSize: 8 })
    const u3 = q.enqueue(new File([new Uint8Array(8)], "c.bin"), { key: "c", forceStaged: true, blockSize: 8 })

    await vi.waitFor(() => {
      expect([u1, u2, u3].filter((u) => u.status === "uploading")).toHaveLength(2)
    })
    expect(u3.status).toBe("queued")

    gate.resolve()
    await Promise.all([u1.done, u2.done, u3.done])
    expect(u1.status).toBe("completed")
    expect(u2.status).toBe("completed")
    expect(u3.status).toBe("completed")
  })

  it("supports pauseAll/resumeAll/clearCompleted", async () => {
    const t = new FakeTransport(8, 8)
    const q = new UploadQueue(t, "ns", { maxConcurrent: 1 })
    const u1 = q.enqueue(new File([new Uint8Array(8)], "a.bin"), { key: "a", forceStaged: true, blockSize: 8 })
    const u2 = q.enqueue(new File([new Uint8Array(8)], "b.bin"), { key: "b", forceStaged: true, blockSize: 8 })

    await Promise.all([u1.done, u2.done])
    expect(u1.status).toBe("completed")
    expect(u2.status).toBe("completed")

    const summary = q.summary()
    expect(summary.done).toBe(2)
    expect(summary.active).toBe(0)

    q.clearCompleted()
    expect(q.all()).toHaveLength(0)
  })

  it("throws when options.key is missing", () => {
    const q = new UploadQueue(new FakeTransport(8, 8), "ns")
    expect(() => q.enqueue(new File([], "x.bin"), {})).toThrow("options.key is required")
  })

  it("restores paused uploads from persistence and resumes them", async () => {
    const backing = memPersistence<{ jobs: Record<string, UploadJobSnapshot> }>()
    const persistence = new UploadJobStore(backing)
    const t = new FakeTransport(20, 8)
    t.hangFirstChunk = true

    const u = new ResumableUpload(t, testFile(), "abc", {
      key: "abc",
      forceStaged: true,
      blockSize: 8,
      concurrency: 1,
    }, persistence)
    u.start()
    await vi.waitFor(() => expect(u.status).toBe("uploading"))
    u.pause()
    await vi.waitFor(() => expect(u.status).toBe("paused"))

    const snapshots = await persistence.load()
    expect(snapshots).toHaveLength(1)
    expect(snapshots[0]!.status).toBe("paused")
    expect(snapshots[0]!.sessionId).toBe("s1")
    expect(snapshots[0]!.blockHashes).toHaveLength(3)

    const q2 = new UploadQueue(t, "ns", { persistence })
    const restored = await q2.restoreAsync()
    expect(restored).toHaveLength(1)
    expect(restored[0]!.status).toBe("paused")

    restored[0]!.resume()
    const doc = await restored[0]!.done
    expect(doc?._id_).toBe("abc")
    expect(restored[0]!.status).toBe("completed")

    expect(await persistence.load()).toHaveLength(0)
  })
})

describe("UploadJobStore", () => {
  it("round-trips snapshots through the backing persistence", async () => {
    const backing = memPersistence<{ jobs: Record<string, UploadJobSnapshot> }>()
    const store = new UploadJobStore(backing)

    await store.save({
      id: "u1",
      key: "a",
      name: "a.bin",
      size: 8,
      type: "application/octet-stream",
      sessionId: "s1",
      blockSize: 8,
      blockHashes: [],
      fingerprint: "",
      uploadedBytes: 0,
      status: "paused",
      error: null,
    })
    await store.save({
      id: "u2",
      key: "b",
      name: "b.bin",
      size: 8,
      type: "application/octet-stream",
      sessionId: null,
      blockSize: 0,
      blockHashes: [],
      fingerprint: "",
      uploadedBytes: 0,
      status: "queued",
      error: null,
    })

    expect(await store.load()).toHaveLength(2)
    await store.remove("u1")
    const rest = await store.load()
    expect(rest).toHaveLength(1)
    expect(rest[0]!.id).toBe("u2")
  })

  it("hydrates existing state from the backing store", async () => {
    const backing = memPersistence<{ jobs: Record<string, UploadJobSnapshot> }>()
    const snap: UploadJobSnapshot = {
      id: "u9",
      key: "x",
      name: "x.bin",
      size: 8,
      type: "application/octet-stream",
      sessionId: "s9",
      blockSize: 8,
      blockHashes: [],
      fingerprint: "",
      uploadedBytes: 0,
      status: "paused",
      error: null,
    }
    await backing.set("x", { jobs: { u9: snap } })

    const store = new UploadJobStore(backing)
    expect(await store.load()).toHaveLength(1)
    expect((await store.load())[0]!.id).toBe("u9")
  })

  it("behaves as memory-only when no backing store is given", async () => {
    const store = new UploadJobStore()
    expect(await store.load()).toEqual([])
    await store.save({
      id: "u1",
      key: "a",
      name: "a.bin",
      size: 8,
      type: "application/octet-stream",
      sessionId: null,
      blockSize: 8,
      blockHashes: [],
      fingerprint: "",
      uploadedBytes: 0,
      status: "paused",
      error: null,
    })
    expect(await store.load()).toHaveLength(1)
    await store.remove("u1")
    expect(await store.load()).toEqual([])
  })

  it("accepts any UploadPersistence implementation in the engine", async () => {
    let saveCount = 0
    let removeCount = 0
    const custom: UploadPersistence = {
      save: () => void saveCount++,
      load: () => [],
      remove: () => void removeCount++,
    }
    const t = new FakeTransport(8, 8)
    const u = new ResumableUpload(t, testFile(), "abc", {
      key: "abc",
      forceStaged: true,
      blockSize: 8,
      concurrency: 1,
    }, custom)
    u.start()
    await u.done

    expect(saveCount).toBeGreaterThan(0)
    expect(removeCount).toBe(1)
  })
})

describe("HestiaBlobClient", () => {
  let blobs: HestiaBlobClient
  let client: HttpTransport
  let raw: any

  beforeEach(() => {
    const provider = makeProvider()
    client = new HttpTransport("http://test.local", "/api")
    const mock = (createNetworkClient as ReturnType<typeof vi.fn>).mock
    raw = mock.results[mock.results.length - 1]!.value
    vi.clearAllMocks()
    blobs = new HestiaBlobClient(client, "/api")
  })

  describe("blob (download URL)", () => {
    it("composes download url", () => {
      const url = blobs.blob("test-bucket", "b1")
      expect(url).toBe("http://test.local/api/system/blobs/blob/download/test-bucket/b1")
    })

    it("composes download url from custom baseUrl and prefix", () => {
      const customClient = new HttpTransport("http://other.local:9090", "/prefix")
      const customBlobs = new HestiaBlobClient(customClient, "/prefix")
      const url = customBlobs.blob("custom", "x")
      expect(url).toBe("http://other.local:9090/prefix/system/blobs/blob/download/custom/x")
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

describe("HttpTransport URL composition", () => {
  it("combines baseUrl, prefix and path", async () => {
    const provider = makeProvider()
    const client = new HttpTransport("http://example.com", "/v2")
    const mock = (createNetworkClient as ReturnType<typeof vi.fn>).mock
    const raw = mock.results[mock.results.length - 1]!.value

    raw.get.mockResolvedValueOnce(okResponse({ data: {} }))
    await client.get("/system/health")
    expect(raw.get).toHaveBeenCalledWith("v2/system/health", { headers: {} })
  })
})

describe("HttpTransport stream", () => {
  let client: HttpTransport

  beforeEach(() => {
    const provider = makeProvider()
    client = new HttpTransport("http://test.local", "/api")
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
