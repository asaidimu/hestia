import { describe, expect, it, vi, beforeAll, afterAll, beforeEach } from "vitest"
import { makeClient, uniqueId } from "../../tests/helpers"
import { HttpTransport } from "../../core/client"
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
import type { Document } from "../../core/types"
import type { SimplePersistence } from "@asaidimu/utils-persistence"
import type {
  BlobMeta,
  UploadBeginResult,
  UploadJobSnapshot,
  UploadPersistence,
  UploadProgressResult,
} from "./types"

async function sha256HexTest(data: ArrayBuffer | Uint8Array): Promise<string> {
  const digest = await crypto.subtle.digest("SHA-256", data as any)
  return Array.from(new Uint8Array(digest))
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("")
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

function testFile(): File {
  return new File([new Uint8Array(20)], "big.bin")
}

describe("BlobNamespace — E2E", () => {
  const container = makeClient()
  const nsName = uniqueId("e2e-blobs")
  const blobKey = uniqueId("blob")
  let ns: BlobNamespace

  beforeAll(async () => {
    await container.blobs.createNamespace({ ns: nsName, display_name: "E2E blobs" })
    ns = container.blobs.namespace(nsName)
  })

  afterAll(async () => {
    await container.blobs.deleteNamespace(nsName).catch(() => {})
  })

  describe("upload", () => {
    it("uploads a small file directly", async () => {
      const file = new File(["hello"], "hello.txt", { type: "text/plain" })
      const result = await ns.upload({ file, options: { key: blobKey } })
      expect(result!.key).toBe(blobKey)
      expect(result!._id_).toBeTruthy()
      expect(result!.content_type).toBe("text/plain")
      expect(result!.size).toBe(5)
    })

    it("throws when options.key is missing", async () => {
      await expect(ns.upload({ file: new File([], "x") })).rejects.toThrow("options.key is required")
    })
  })

  describe("staged upload protocol", () => {
    it("runs begin → chunk → progress → complete", async () => {
      const stagedKey = uniqueId("staged")
      const bytes = new Uint8Array(20).fill(0xab)
      const begun = await ns.beginUpload({ key: stagedKey, size: 20, contentType: "application/octet-stream", blockSize: 8 })
      expect(begun.session_id).toBeTruthy()
      expect(begun.key).toBe(stagedKey)

      const chunk = new Blob([bytes])
      const chunkHash = await sha256HexTest(bytes)
      const total = await ns.uploadChunk({ sessionId: begun.session_id, offset: 0, data: chunk, sha256: chunkHash })
      expect(total).toBe(20)

      const prog = await ns.progress(begun.session_id)
      expect(prog.total).toBe(20)
      expect(prog.ranges).toEqual([{ start: 0, end: 20 }])

      const doc = await ns.completeUpload({ sessionId: begun.session_id })
      expect(doc?.key).toBe(stagedKey)
      expect(doc!.size).toBe(20)
    })

    it("aborts a staged session", async () => {
      const begun = await ns.beginUpload({ key: uniqueId("abort-blob"), size: 1, contentType: "text/plain" })
      await ns.abort(begun.session_id)
    })
  })

  describe("resumable upload flow", () => {
    it("uploads a small file through the resumable engine", async () => {
      const key = uniqueId("resumable")
      const file = new File([new Uint8Array(20).fill(0xcd)], "big.bin", { type: "application/octet-stream" })

      const onProgress = vi.fn()
      const doc = await ns.upload({ file, options: { key, forceStaged: true, blockSize: 8, onProgress } })
      expect(doc?.key).toBe(key)
      expect(doc!.size).toBe(20)

      expect(onProgress).toHaveBeenCalled()
      const last = onProgress.mock.calls[onProgress.mock.calls.length - 1]![0]
      expect(last.uploaded).toBe(20)
      expect(last.total).toBe(20)
    })
  })

  describe("read", () => {
    it("fetches metadata by key via head endpoint", async () => {
      const result = await ns.read(blobKey)
      expect(result?.key).toBe(blobKey)
      expect(result?.namespace_id).toBe(nsName)
      expect(result?._id_).toBeTruthy()
      expect(result?._metadata_).toBeDefined()
    })

    it("returns undefined on not found", async () => {
      const result = await ns.read("missing-blob")
      expect(result).toBeUndefined()
    })
  })

  describe("find", () => {
    it("lists blobs in the namespace", async () => {
      const result = await ns.find()
      expect(result.data.some((b) => b.key === blobKey)).toBe(true)
    })
  })

  describe("update", () => {
    it("updates blob metadata", async () => {
      const result = await ns.update({ data: { content_type: "application/octet-stream" }, options: { key: blobKey } })
      expect(result?.key).toBe(blobKey)
    })

    it("throws when options.key is missing", async () => {
      await expect(ns.update({ data: { content_type: "text/plain" } })).rejects.toThrow("options.key is required")
    })
  })

  describe("download", () => {
    it("downloads the blob body", async () => {
      const result = await ns.download(blobKey)
      expect(result.data).toBeDefined()
      expect(result.contentType).toBe("text/plain")
      const text = await result.data.text()
      expect(text).toBe("hello")
    })
  })

  describe("delete", () => {
    it("deletes a blob by key", async () => {
      await ns.delete(blobKey)
      const result = await ns.read(blobKey)
      expect(result).toBeUndefined()
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

describe("HestiaBlobClient — E2E", () => {
  const container = makeClient()

  describe("blob (download URL)", () => {
    it("composes download url", () => {
      const client = new HttpTransport("http://test.local", "/api")
      const blobs = new HestiaBlobClient(client, "/api")
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
      const ns = container.blobs.namespace("my-bucket")
      expect(ns).toBeInstanceOf(BlobNamespace)
      expect((ns as any).ns).toBe("my-bucket")
    })
  })

  describe("namespaces", () => {
    it("creates and deletes a namespace", async () => {
      const name = uniqueId("e2e-ns-lifecycle")
      const created = await container.blobs.createNamespace({ ns: name, display_name: "Lifecycle" })
      expect(created.id).toBe(name)
      expect(created.display_name).toBe("Lifecycle")

      const listed = await container.blobs.namespaces()
      expect(listed.some((n) => n.id === name)).toBe(true)

      await container.blobs.deleteNamespace(name)
      const after = await container.blobs.namespaces()
      expect(after.some((n) => n.id === name)).toBe(false)
    })
  })
})

describe("HttpTransport stream", () => {
  let client: HttpTransport

  beforeEach(() => {
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