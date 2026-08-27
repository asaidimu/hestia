import type { QueryDSL } from "@asaidimu/query";
import type { SimplePersistence } from "@asaidimu/utils-persistence";
import { ReactiveDataStore } from "@asaidimu/utils-store";
import { type Transport } from "../../core/client";
import { createPagedController } from "../../core/pager";
import type { Document, Page, PagedData, StoreEvent } from "../../core/types";
import type { DocumentStore } from "../../core/types";
import type {
    BlobDocument,
    BlobMeta,
    CompactResult,
    CreateNamespaceRequest,
    ListBlobsRequest,
    NamespaceInfo,
    NamespaceStats,
    UploadBeginResult,
    UploadJobSnapshot,
    UploadJobsState,
    UploadOptions,
    UploadPersistence,
    UploadProgressResult,
    UploadQueueOptions,
    UploadStatus,
} from "./types";

// ── Tunables (mirror examples/staging client) ────────────────────────────────
const DEFAULT_BLOCK_SIZE = 8 << 20; // 8 MiB (GOOD_BLOCK_SIZE)
const DEFAULT_DIRECT_THRESHOLD = 16 << 20; // 16 MiB
const DEFAULT_CONCURRENCY = 4;
const DEFAULT_RETRIES = 3;
const MAX_CONCURRENCY = 6;
const MIN_CONCURRENCY = 1;
const AIMD_SUCCESS_STREAK = 3;
const AIMD_BACKOFF_COOLDOWN_MS = 2500;
const EWMA_ALPHA = 0.3;
const RETRY_DELAY_MS = 1500;
const INITIAL_THROUGHPUT_BPS = 256 * 1024;
const RENDER_THROTTLE_MS = 80;

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function sha256Hex(buffer: ArrayBuffer | ArrayBufferView): Promise<string> {
  const digest = await crypto.subtle.digest("SHA-256", buffer as any);
  const bytes = new Uint8Array(digest);
  let hex = "";
  for (const b of bytes) hex += b.toString(16).padStart(2, "0");
  return hex;
}

// Bun's File/Blob type declarations omit `slice` (runtime has it), so cast.
interface SlicableFile {
  slice(start?: number, end?: number, contentType?: string): Blob;
}

function sliceFile(file: File, start: number, end: number): Blob {
  return (file as unknown as SlicableFile).slice(start, end);
}

function genId(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return "upload_" + Date.now() + "_" + Math.random().toString(36).slice(2);
}

// getMissingRanges turns the server's received ranges into the gaps that
// still need uploading (same contract as the prototype client).
export function getMissingRanges(
  ranges: { start: number; end: number }[],
  fileSize: number,
): { start: number; end: number }[] {
  const missing: { start: number; end: number }[] = [];
  let expected = 0;
  for (const r of ranges) {
    if (r.start > expected) missing.push({ start: expected, end: r.start });
    if (r.end > expected) expected = r.end;
  }
  if (expected < fileSize) missing.push({ start: expected, end: fileSize });
  return missing;
}

// createChunkCursor walks missing ranges, yielding blockSize-aligned slices
// (faithful port of the prototype's chunk cursor).
function createChunkCursor(
  missingRanges: { start: number; end: number }[],
  blockSize: number,
): {
  hasMore: () => boolean;
  next: () => { start: number; end: number } | null;
} {
  let ri = 0;
  let pos = missingRanges.length ? missingRanges[0]!.start : 0;
  function advance() {
    while (ri < missingRanges.length && pos >= missingRanges[ri]!.end) {
      ri++;
      if (ri < missingRanges.length) pos = missingRanges[ri]!.start;
    }
  }
  return {
    hasMore() {
      advance();
      return ri < missingRanges.length;
    },
    next() {
      advance();
      if (ri >= missingRanges.length) return null;
      const end = Math.min(pos + blockSize, missingRanges[ri]!.end);
      const range = { start: pos, end };
      pos = end;
      return range;
    },
  };
}

// AimdController adjusts the in-flight chunk window AIMD-style and keeps an
// EWMA throughput estimate (faithful port of the prototype's net controller).
export class AimdController {
  maxConcurrency: number;
  consecutiveSuccesses = 0;
  lastBackoff = 0;
  throughputEstimate: number;

  constructor(maxConcurrency: number, initialThroughput = INITIAL_THROUGHPUT_BPS) {
    this.maxConcurrency = maxConcurrency;
    this.throughputEstimate = initialThroughput;
  }

  estThroughput(): number {
    return this.throughputEstimate;
  }

  recordThroughput(bps: number): void {
    this.throughputEstimate =
      this.throughputEstimate * (1 - EWMA_ALPHA) + bps * EWMA_ALPHA;
  }

  recordSuccess(): void {
    this.consecutiveSuccesses++;
    if (this.consecutiveSuccesses >= AIMD_SUCCESS_STREAK) {
      this.consecutiveSuccesses = 0;
      this.maxConcurrency = Math.min(MAX_CONCURRENCY, this.maxConcurrency + 1);
    }
  }

  recordFailure(): void {
    const now = Date.now();
    if (now - this.lastBackoff < AIMD_BACKOFF_COOLDOWN_MS) return;
    this.lastBackoff = now;
    this.consecutiveSuccesses = 0;
    this.maxConcurrency = Math.max(MIN_CONCURRENCY, Math.floor(this.maxConcurrency / 2));
  }
}

/**
 * UploadJobStore adapts a `SimplePersistence<UploadJobsState>` into the
 * engine's `UploadPersistence` interface. Snapshots are stored keyed by
 * upload id; the backing store is optional and defaults to memory-only
 * (uploads are not resumed across sessions when omitted).
 */
export class UploadJobStore implements UploadPersistence {
  private state: UploadJobsState = { jobs: {} };
  private readonly instanceId = genId();
  private readonly ready: Promise<void>;

  constructor(private readonly backing?: SimplePersistence<UploadJobsState>) {
    this.ready = this.hydrate();
  }

  private async hydrate(): Promise<void> {
    if (!this.backing) return;
    const existing = await this.backing.get();
    if (existing) this.state = existing;
  }

  async save(snapshot: UploadJobSnapshot): Promise<void> {
    await this.ready;
    this.state = { jobs: { ...this.state.jobs, [snapshot.id]: snapshot } };
    await this.backing?.set(this.instanceId, this.state);
  }

  async load(): Promise<UploadJobSnapshot[]> {
    await this.ready;
    if (this.backing) {
      const existing = await this.backing.get();
      if (existing) this.state = existing;
    }
    return Object.values(this.state.jobs);
  }

  async remove(id: string): Promise<void> {
    await this.ready;
    const jobs = { ...this.state.jobs };
    delete jobs[id];
    this.state = { jobs };
    await this.backing?.set(this.instanceId, this.state);
  }
}

/** The staged-upload transport methods ResumableUpload drives. */
export interface UploadTransport {
  beginUpload(props: {
    key: string;
    size: number;
    contentType?: string;
    blockSize?: number;
    overwrite?: boolean;
  }): Promise<UploadBeginResult>;
  uploadChunk(props: {
    sessionId: string;
    offset: number;
    data: Blob;
    sha256?: string;
    signal?: AbortSignal;
  }): Promise<number>;
  completeUpload(props: { sessionId: string; overwrite?: boolean }): Promise<Document<BlobMeta> | undefined>;
  progress(sessionId: string): Promise<UploadProgressResult>;
  abort(sessionId: string): Promise<void>;
}

type ChunkResult = { success: boolean; aborted: boolean };
type PoolResult = { paused: boolean; cancelled: boolean };

/**
 * ResumableUpload drives one staged blob upload end-to-end: streaming
 * pre-hash, session begin, missing-range resume, AIMD-bounded worker pool,
 * and completion. It is a faithful port of the prototype's job object.
 *
 * Lifecycle is explicit: construct in "queued", call `start()` to begin;
 * `pause()`/`resume()`/`cancel()`/`retry()` control it from there.
 *
 * `done` resolves when the *current attempt* reaches a terminal state
 * (completed → document, cancelled → undefined, error → rejection). It is a
 * getter over the live attempt, so after `retry()`/`resume()` from an error,
 * re-awaiting `done` yields the outcome of the new attempt instead of the
 * stale rejected promise.
 */
export class ResumableUpload {
  readonly id: string;
  status: UploadStatus = "queued";
  blockSize = 0;
  sessionId: string | null = null;
  uploaded = 0;
  hashed = 0;
  hashTotal = 0;
  bytesPerSecond = 0;
  etaSeconds = 0;
  fingerprint = "";
  error: Error | null = null;

  /** Internal hook (set by UploadQueue) fired on every status change. */
  onStatusChange?: (status: UploadStatus) => void;

  /**
   * Promise for the current attempt. Re-read after `retry()`/`resume()` to
   * await the restarted attempt rather than the settled one from before.
   */
  get done(): Promise<Document<BlobMeta> | undefined> {
    return this.attempt;
  }

  readonly total: number;
  private readonly key: string;
  private readonly file: File;
  private readonly options: UploadOptions;
  private readonly transport: UploadTransport;
  private readonly persistence?: UploadPersistence;
  private readonly aimd: AimdController;
  private readonly startedAt = Date.now();

  private blockHashes: string[] = [];
  private pauseRequested = false;
  private cancelRequested = false;
  private abortControllers = new Set<AbortController>();
  private running = false;
  private lastProgressTime = 0;
  private lastProgressBytes = 0;
  private speedSamples: number[] = [];
  private lastReportAt = 0;
  private attempt!: Promise<Document<BlobMeta> | undefined>;
  private resolveDone!: (doc: Document<BlobMeta> | undefined) => void;
  private rejectDone!: (err: Error) => void;

  constructor(
    transport: UploadTransport,
    file: File,
    key: string,
    options: UploadOptions,
    persistence?: UploadPersistence,
    resumeFrom?: UploadJobSnapshot,
  ) {
    this.transport = transport;
    this.file = file;
    this.key = key;
    this.options = options;
    this.persistence = persistence;
    this.id = resumeFrom?.id ?? genId();
    this.total = resumeFrom?.size ?? file.size;
    this.blockSize = resumeFrom?.blockSize ?? 0;
    this.sessionId = resumeFrom?.sessionId ?? null;
    this.blockHashes = resumeFrom?.blockHashes ?? [];
    this.fingerprint = resumeFrom?.fingerprint ?? "";
    this.uploaded = resumeFrom?.uploadedBytes ?? 0;
    this.status = resumeFrom ? "paused" : "queued";
    this.aimd = new AimdController(Math.max(1, options.concurrency ?? DEFAULT_CONCURRENCY));
    this.newAttempt();
    if (options.signal) {
      if (options.signal.aborted) {
        queueMicrotask(() => void this.cancel());
      } else {
        options.signal.addEventListener("abort", () => void this.cancel(), { once: true });
      }
    }
  }

  private newAttempt(): void {
    this.attempt = new Promise<Document<BlobMeta> | undefined>((res, rej) => {
      this.resolveDone = res;
      this.rejectDone = rej;
    });
  }

  start(): void {
    if (this.status !== "queued") return;
    this.pauseRequested = false;
    this.cancelRequested = false;
    void this.run();
  }

  resume(): void {
    if (this.status !== "paused" && this.status !== "error") return;
    if (this.status === "error") this.newAttempt();
    this.error = null;
    this.pauseRequested = false;
    this.cancelRequested = false;
    void this.run();
  }

  retry(): void {
    this.resume();
  }

  pause(): void {
    if (this.status === "completed" || this.status === "cancelled") return;
    this.pauseRequested = true;
    for (const ac of this.abortControllers) ac.abort();
    if (!this.running) {
      this.setStatus("paused");
      this.report();
      void this.persist();
    }
  }

  async cancel(): Promise<void> {
    if (this.status === "completed" || this.status === "cancelled") return;
    this.cancelRequested = true;
    for (const ac of this.abortControllers) ac.abort();
    if (this.sessionId) {
      try {
        await this.transport.abort(this.sessionId);
      } catch {
        // best-effort cleanup
      }
    }
    await this.persistence?.remove(this.id);
    if (!this.running) {
      this.setStatus("cancelled");
      this.report();
      this.resolveDone(undefined);
    }
    // If running, the run() loop observes cancelRequested and finalizes.
  }

  // ── Orchestrator (port of the prototype's runJob) ─────────────────────────

  private async run(): Promise<void> {
    if (this.running || this.status === "completed" || this.status === "cancelled") return;
    this.running = true;
    try {
      if (this.blockSize === 0) {
        this.blockSize = this.chooseBlockSize();
        await this.persist();
      }

      if (!this.sessionId) {
        const begun = await this.transport.beginUpload({
          key: this.key,
          size: this.total,
          contentType: this.options.contentType || this.file.type,
          blockSize: this.blockSize,
          overwrite: this.options.overwrite,
        });
        this.sessionId = begun.session_id;
        if (this.blockHashes.length === 0 && begun.block_size > 0) {
          this.blockSize = begun.block_size;
        }
        await this.persist();
      }
      if (this.pauseRequested) {
        this.setStatus("paused");
        await this.persist();
        return;
      }

      if (this.blockHashes.length === 0 && this.blockSize > 0) {
        this.setStatus("hashing");
        await this.hashBlocks();
      }
      if (this.pauseRequested) {
        this.setStatus("paused");
        await this.persist();
        return;
      }
      if (this.cancelRequested) {
        this.finishCancelled();
        return;
      }

      this.setStatus("uploading");
      while (!this.cancelRequested) {
        if (this.pauseRequested) break;
        if (!this.sessionId) throw new Error("upload session missing");
        const prog = await this.transport.progress(this.sessionId);
        this.uploaded = prog.total;
        this.report();
        if (this.uploaded >= this.total) break;

        const missing = getMissingRanges(prog.ranges ?? [], this.total);
        if (missing.length === 0) break;

        const result = await this.runUploadPool(missing);
        if (result.cancelled) break;
        if (result.paused) {
          this.setStatus("paused");
          this.report();
          await this.persist();
          return;
        }
      }

      if (this.cancelRequested) {
        this.finishCancelled();
        return;
      }

      if (!this.sessionId) throw new Error("upload session missing");
      this.setStatus("completing");
      this.report();
      const doc = await this.transport.completeUpload({
        sessionId: this.sessionId,
        overwrite: this.options.overwrite,
      });
      this.uploaded = this.total;
      this.setStatus("completed");
      this.report();
      await this.persistence?.remove(this.id);
      this.resolveDone(doc);
    } catch (err) {
      if (this.cancelRequested) {
        this.finishCancelled();
        return;
      }
      if (this.pauseRequested) {
        this.setStatus("paused");
        await this.persist();
        return;
      }
      this.error = err instanceof Error ? err : new Error(String(err));
      this.setStatus("error");
      this.report();
      await this.persist();
      this.rejectDone(this.error);
    } finally {
      this.running = false;
    }
  }

  private finishCancelled(): void {
    this.setStatus("cancelled");
    this.report();
    void this.persistence?.remove(this.id);
    this.resolveDone(undefined);
  }

  private chooseBlockSize(): number {
    const preferred = this.options.blockSize ?? DEFAULT_BLOCK_SIZE;
    if (this.total <= 0) return 0;
    return Math.min(preferred, this.total);
  }

  // ── Streaming pre-hash (one block in memory at a time) ────────────────────

  private async hashBlocks(): Promise<void> {
    const blockSize = this.blockSize;
    const totalBlocks = Math.ceil(this.total / blockSize);
    this.blockHashes = new Array<string>(totalBlocks);
    this.hashTotal = this.total;
    this.hashed = 0;
    let lastYield = performance.now();

    for (let i = 0; i < totalBlocks; i++) {
      if (this.cancelRequested) throw new Error("Pre-hashing cancelled");
      while (this.pauseRequested && !this.cancelRequested) await sleep(200);
      if (this.cancelRequested) throw new Error("Pre-hashing cancelled");

      const start = i * blockSize;
      const end = Math.min(start + blockSize, this.total);
      const slice = sliceFile(this.file, start, end);
      const buf = await slice.arrayBuffer();
      this.blockHashes[i] = await sha256Hex(buf);
      this.hashed = end;
      this.report();

      const now = performance.now();
      if (now - lastYield > 12) {
        await sleep(0);
        lastYield = now;
      }
    }

    const concatenated = this.blockHashes.join("");
    this.fingerprint = await sha256Hex(new TextEncoder().encode(concatenated));
    await this.persist();
  }

  // ── Upload pool (port of the prototype's runUploadPool) ───────────────────

  private runUploadPool(missing: { start: number; end: number }[]): Promise<PoolResult> {
    return new Promise((resolve) => {
      const cursor = createChunkCursor(missing, this.blockSize);
      let finished = false;
      let permanentFailure: Error | null = null;
      let sawPause = false;
      let activeWorkers = 0;

      const finishIfDone = () => {
        if (finished || activeWorkers > 0) return;
        finished = true;
        if (permanentFailure) {
          void this.failPool(permanentFailure);
          resolve({ paused: false, cancelled: false });
        } else {
          resolve({ paused: sawPause && !this.cancelRequested, cancelled: this.cancelRequested });
        }
      };

      const spawn = () => {
        activeWorkers++;
        void (async () => {
          while (true) {
            if (this.cancelRequested) break;
            if (this.pauseRequested) {
              sawPause = true;
              break;
            }
            if (permanentFailure) break;
            if (activeWorkers > this.aimd.maxConcurrency) break;
            const range = cursor.next();
            if (!range) break;
            const result = await this.uploadOneChunk(range);
            if (!result.success) {
              if (!result.aborted) permanentFailure = new Error(`Chunk failed at ${range.start}`);
              else if (this.pauseRequested) sawPause = true;
              break;
            }
          }
        })().finally(() => {
          activeWorkers--;
          if (
            !this.cancelRequested &&
            !this.pauseRequested &&
            !permanentFailure &&
            activeWorkers < this.aimd.maxConcurrency &&
            cursor.hasMore()
          ) {
            spawn();
          }
          finishIfDone();
        });
      };

      const initialWorkers = Math.max(1, this.aimd.maxConcurrency);
      let spawned = 0;
      for (let i = 0; i < initialWorkers && cursor.hasMore(); i++) {
        spawn();
        spawned++;
      }
      if (spawned === 0) finishIfDone();
    });
  }

  private failPool(err: Error): void {
    if (this.status === "completed" || this.status === "cancelled") return;
    this.error = err;
    this.setStatus("error");
    this.report();
    void this.persist();
    this.rejectDone(this.error);
  }

  private async uploadOneChunk(range: { start: number; end: number }): Promise<ChunkResult> {
    const chunk = sliceFile(this.file, range.start, range.end);
    if (this.cancelRequested || this.pauseRequested) return { success: false, aborted: true };

    const blockIndex = Math.floor(range.start / this.blockSize);
    const chunkHash = this.blockHashes[blockIndex];
    const maxRetries = Math.max(0, this.options.retries ?? DEFAULT_RETRIES);
    const chunkBytes = range.end - range.start;

    for (let attempt = 1; attempt <= maxRetries + 1; attempt++) {
      if (this.cancelRequested || this.pauseRequested) return { success: false, aborted: true };

      const ac = new AbortController();
      this.abortControllers.add(ac);
      const estBps = Math.max(this.aimd.estThroughput(), 32 * 1024);
      const timeoutMs = Math.max(6000, (chunkBytes / estBps) * 1000 * 3);
      const timeoutId = setTimeout(() => ac.abort(), timeoutMs);

      const t0 = performance.now();
      try {
        if (!this.sessionId) throw new Error("upload session missing");
        const total = await this.transport.uploadChunk({
          sessionId: this.sessionId,
          offset: range.start,
          data: chunk,
          sha256: chunkHash,
          signal: ac.signal,
        });
        clearTimeout(timeoutId);
        this.abortControllers.delete(ac);
        const elapsed = (performance.now() - t0) / 1000;
        this.aimd.recordSuccess();
        this.onChunkUploaded(chunkBytes, elapsed, total);
        return { success: true, aborted: false };
      } catch (err) {
        clearTimeout(timeoutId);
        this.abortControllers.delete(ac);
        if (this.cancelRequested) return { success: false, aborted: true };
        if (this.pauseRequested && err instanceof DOMException && err.name === "AbortError") {
          return { success: false, aborted: true };
        }
        this.aimd.recordFailure();
        if (attempt <= maxRetries) {
          await sleep(RETRY_DELAY_MS * Math.pow(2, attempt - 1));
        }
      }
    }
    return { success: false, aborted: false };
  }

  private onChunkUploaded(bytes: number, elapsedSec: number, serverTotal: number): void {
    if (elapsedSec > 0.05) {
      this.aimd.recordThroughput(bytes / elapsedSec);
    }
    this.uploaded = serverTotal;

    const now = Date.now();
    if (!this.lastProgressTime) {
      this.lastProgressTime = now;
      this.lastProgressBytes = serverTotal;
    } else if (now - this.lastProgressTime > 1000) {
      const deltaBytes = serverTotal - this.lastProgressBytes;
      const deltaSec = (now - this.lastProgressTime) / 1000;
      if (deltaSec > 0) {
        const speed = deltaBytes / deltaSec;
        this.speedSamples.push(speed);
        if (this.speedSamples.length > 5) this.speedSamples.shift();
      }
      this.lastProgressTime = now;
      this.lastProgressBytes = serverTotal;
    }

    const avg =
      this.speedSamples.length > 0
        ? this.speedSamples.reduce((a, b) => a + b, 0) / this.speedSamples.length
        : 0;
    this.bytesPerSecond = avg;
    const remaining = this.total - this.uploaded;
    this.etaSeconds = avg > 0 && remaining > 0 ? remaining / avg : 0;
    this.report();
  }

  // ── Progress reporting & persistence ──────────────────────────────────────

  private report(): void {
    const now = performance.now();
    const force = this.status !== "uploading";
    if (!force && now - this.lastReportAt < RENDER_THROTTLE_MS) return;
    this.lastReportAt = now;

    const phase = this.status === "hashing" ? "hash" : "upload";
    const percent =
      phase === "hash"
        ? this.hashTotal > 0
          ? this.hashed / this.hashTotal
          : 0
        : this.total > 0
          ? this.uploaded / this.total
          : 1;
    this.options.onProgress?.({
      phase,
      hashed: this.hashed,
      hashTotal: this.hashTotal,
      uploaded: this.uploaded,
      total: this.total,
      percent: Math.min(1, percent),
      bytesPerSecond: this.bytesPerSecond,
      etaSeconds: this.etaSeconds,
      fingerprint: this.fingerprint || undefined,
    });
  }

  private snapshot(): UploadJobSnapshot {
    return {
      id: this.id,
      key: this.key,
      name: this.file.name,
      size: this.total,
      type: this.file.type,
      file: this.file,
      sessionId: this.sessionId,
      blockSize: this.blockSize,
      blockHashes: this.blockHashes,
      fingerprint: this.fingerprint,
      uploadedBytes: this.uploaded,
      status: this.status,
      error: this.error ? this.error.message : null,
    };
  }

  private async persist(): Promise<void> {
    if (!this.persistence) return;
    try {
      await this.persistence.save(this.snapshot());
    } catch {
      // best-effort
    }
  }

  private setStatus(s: UploadStatus): void {
    if (this.status === s) return;
    this.status = s;
    this.onStatusChange?.(s);
  }
}

/**
 * UploadQueue schedules multiple ResumableUploads with a concurrency cap,
 * mirroring the prototype's job queue (MAX_CONCURRENT_JOBS = 3).
 */
export class UploadQueue {
  private jobs = new Map<string, ResumableUpload>();
  private queue: string[] = [];
  private active = 0;
  private runningJobs = new WeakSet<ResumableUpload>();
  private readonly maxConcurrent: number;
  private readonly persistence?: UploadPersistence;
  private readonly transport: UploadTransport;
  private readonly ns: string;

  constructor(
    transport: UploadTransport,
    ns: string,
    options: UploadQueueOptions = {},
  ) {
    this.transport = transport;
    this.ns = ns;
    this.maxConcurrent = Math.max(1, options.maxConcurrent ?? 3);
    this.persistence = options.persistence;
  }

  enqueue(file: File, options: UploadOptions): ResumableUpload {
    const key = options.key as string;
    if (!key) throw new Error("options.key is required for blob upload");
    const upload = new ResumableUpload(this.transport, file, key, options, this.persistence);
    upload.onStatusChange = (s) => {
      if (s === "queued") return;
      const running = s === "hashing" || s === "uploading" || s === "completing";
      const suspended = s === "paused" || s === "completed" || s === "error" || s === "cancelled";
      if (running && !this.runningJobs.has(upload)) {
        this.runningJobs.add(upload);
        this.active++;
        this.pump();
      } else if (suspended && this.runningJobs.has(upload)) {
        this.runningJobs.delete(upload);
        this.active = Math.max(0, this.active - 1);
        this.pump();
      }
      if (s === "cancelled" || s === "completed") {
        const idx = this.queue.indexOf(upload.id);
        if (idx !== -1) this.queue.splice(idx, 1);
      }
    };
    this.jobs.set(upload.id, upload);
    this.queue.push(upload.id);
    this.pump();
    return upload;
  }

  private pump(): void {
    while (this.active < this.maxConcurrent && this.queue.length > 0) {
      const id = this.queue.shift();
      if (!id) continue;
      const upload = this.jobs.get(id);
      if (!upload) continue;
      this.runningJobs.add(upload);
      this.active++;
      upload.start();
    }
  }

  /** Reconstruct uploads from persisted snapshots, paused and ready to resume. */
  restore(fileProvider?: (snapshot: UploadJobSnapshot) => File | undefined): ResumableUpload[] {
    const snapshots = this.persistence ? this.persistence.load() : Promise.resolve([]);
    const loaded = snapshots instanceof Promise ? null : snapshots;
    void loaded;
    // Restore is async-friendly: callers use restoreAsync for Promise adapters.
    return [];
  }

  async restoreAsync(
    fileProvider?: (snapshot: UploadJobSnapshot) => File | undefined,
  ): Promise<ResumableUpload[]> {
    if (!this.persistence) return [];
    const snapshots = await this.persistence.load();
    const restored: ResumableUpload[] = [];
    for (const snap of snapshots) {
      const file = snap.file ?? fileProvider?.(snap);
      if (!file) continue;
      const upload = new ResumableUpload(
        this.transport,
        file,
        snap.key,
        { key: snap.key, contentType: snap.type, overwrite: false },
        this.persistence,
        snap,
      );
      this.jobs.set(upload.id, upload);
      restored.push(upload);
    }
    return restored;
  }

  pause(id: string): void {
    const upload = this.jobs.get(id);
    if (!upload) return;
    if (upload.status === "queued") {
      const idx = this.queue.indexOf(id);
      if (idx !== -1) this.queue.splice(idx, 1);
    }
    upload.pause();
  }

  resume(id: string): void {
    this.jobs.get(id)?.resume();
  }

  retry(id: string): void {
    this.jobs.get(id)?.retry();
  }

  async cancel(id: string): Promise<void> {
    const upload = this.jobs.get(id);
    if (!upload) return;
    const idx = this.queue.indexOf(id);
    if (idx !== -1) this.queue.splice(idx, 1);
    await upload.cancel();
  }

  pauseAll(): void {
    for (const upload of this.jobs.values()) {
      if (upload.status === "uploading" || upload.status === "hashing" || upload.status === "queued") {
        upload.pause();
      }
    }
  }

  resumeAll(): void {
    for (const upload of this.jobs.values()) {
      if (upload.status === "paused" || upload.status === "error") upload.resume();
    }
  }

  clearCompleted(): void {
    for (const [id, upload] of Array.from(this.jobs)) {
      if (upload.status === "completed" || upload.status === "cancelled") {
        this.jobs.delete(id);
      }
    }
  }

  all(): ResumableUpload[] {
    return Array.from(this.jobs.values());
  }

  summary(): {
    active: number;
    paused: number;
    done: number;
    cancelled: number;
    failed: number;
    totalBytes: number;
    uploadedBytes: number;
    bytesPerSecond: number;
    etaSeconds: number;
  } {
    const active = this.all().filter((j) =>
      ["uploading", "hashing", "queued", "completing"].includes(j.status),
    );
    const paused = this.all().filter((j) => j.status === "paused");
    const done = this.all().filter((j) => j.status === "completed");
    const cancelled = this.all().filter((j) => j.status === "cancelled");
    const failed = this.all().filter((j) => j.status === "error");

    let totalBytes = 0;
    let uploadedBytes = 0;
    let speedSum = 0;
    for (const j of active) {
      totalBytes += j.total;
      uploadedBytes += j.uploaded;
      speedSum += j.bytesPerSecond;
    }
    const remaining = Math.max(0, totalBytes - uploadedBytes);
    const etaSeconds = speedSum > 0 ? remaining / speedSum : 0;

    return {
      active: active.length,
      paused: paused.length,
      done: done.length,
      cancelled: cancelled.length,
      failed: failed.length,
      totalBytes,
      uploadedBytes,
      bytesPerSecond: speedSum,
      etaSeconds,
    };
  }
}

function asDoc(b: BlobMeta): BlobDocument {
  return {
    _id_: b.key,
    _metadata_: {
      checksum: "",
      created: b.created_at,
      updated: b.updated_at ?? b.created_at,
      version: 1,
    },
    ...b,
  };
}

function pageMeta<T extends Record<string, any>>(items: T[]): Page<T>["page"] {
  return {
    number: 1,
    size: items.length,
    count: items.length,
    total: items.length,
    pages: 1,
  };
}

export class BlobNamespace implements DocumentStore<BlobMeta, QueryDSL<BlobMeta>, string, QueryDSL<BlobMeta>, Record<string, unknown>, string, Record<string, any>, Record<string, unknown>, { key: string; contentType?: string }, Record<string, unknown>> {
  private pagerOptions = {};
  private pager: PagedData<BlobMeta>;
  private prefixFilter = "";
  private _queue?: UploadQueue;

  constructor(
    private client: Transport,
    private ns: string,
    private persistence?: UploadPersistence,
  ) {
    this.pager = createPagedController<BlobMeta>(
      `blobs_${ns}`,
      new ReactiveDataStore<any>({}),
      this.pagerOptions,
      (query) => this.find(query),
    );
  }

  name() {
    return this.ns;
  }

  setPrefix(prefix: string) {
    this.prefixFilter = prefix;
  }

  async find(query?: QueryDSL<BlobMeta>): Promise<Page<BlobMeta>> {
    const prefix = this.prefixFilter || (query as any)?.prefix || "";
    const limit =
      (query as any)?.limit ?? query?.pagination?.limit ?? 0;

    const req: ListBlobsRequest = {};
    if (prefix) req.prefix = prefix;
    if (limit) req.limit = limit;

    const res = await this.client.dispatch<{
      data: { blobs: BlobMeta[] };
    }>("system:blobs:blob:list", {
      arguments: { ns: this.ns },
      payload: req,
    });

    const items = res.data?.data?.blobs ?? [];
    return { data: items.map(asDoc), loading: false, page: pageMeta(items), error: undefined };
  }

  async read(key: string): Promise<Document<BlobMeta> | undefined> {
    try {
      const res = await this.client.dispatch<{ data: BlobMeta }>(
        "system:blobs:blob:head",
        { arguments: { ns: this.ns, key } },
      );
      if (!res.data?.data) return undefined;
      return asDoc(res.data.data);
    } catch (err: any) {
      if (err?.code === "NOT_FOUND") return undefined;
      throw err;
    }
  }

  async create(_props: { data: Partial<BlobMeta> }): Promise<Document<BlobMeta> | undefined> {
    throw new Error("Use upload() to create blobs");
  }

  async update(props: { data: Partial<BlobMeta>; options?: Record<string, any> }): Promise<Document<BlobMeta> | undefined> {
    const key = props.options?.key as string;
    if (!key) throw new Error("options.key is required for blob update");
    const res = await this.client.dispatch<{ data: BlobMeta }>(
      "system:blobs:blob:update",
      { arguments: { ns: this.ns, key }, payload: { custom: props.data } },
    );
    return asDoc(res.data!.data);
  }

  async delete(key: string): Promise<void> {
    await this.client.dispatch("system:blobs:blob:delete", {
      arguments: { ns: this.ns, key },
    });
  }

  async rename(oldKey: string, newKey: string): Promise<void> {
    await this.client.dispatch("system:blobs:blob:rename", {
      arguments: { ns: this.ns, key: oldKey },
      payload: { new_key: newKey },
    });
  }

  async stats(): Promise<NamespaceStats> {
    const res = await this.client.dispatch<{ data: NamespaceStats }>(
      "system:blobs:namespace:stats",
      { arguments: { ns: this.ns } },
    );
    return res.data!.data;
  }

  async verify(): Promise<void> {
    await this.client.dispatch("system:blobs:namespace:verify", {
      arguments: { ns: this.ns },
    })
  }

  async compact(): Promise<CompactResult> {
    const res = await this.client.dispatch<{ data: CompactResult }>(
      "system:blobs:namespace:compact",
      { arguments: { ns: this.ns } },
    )
    return res.data!.data
  }

  async list(options?: QueryDSL<BlobMeta>): Promise<Page<BlobMeta>> {
    return this.find(options ?? {});
  }

  async upload(props: { file: File; options?: UploadOptions }): Promise<Document<BlobMeta> | undefined> {
    const key = props.options?.key as string;
    if (!key) throw new Error("options.key is required for blob upload");
    const threshold = props.options?.threshold ?? DEFAULT_DIRECT_THRESHOLD;
    if (!props.options?.forceStaged && props.file.size <= threshold) {
      return this.uploadDirect(props.file, key, props.options);
    }
    const upload = this.createUpload(props.file, props.options);
    upload.start();
    return upload.done;
  }

  /** Create a ResumableUpload controller without starting it. */
  createUpload(file: File, options: UploadOptions = {}): ResumableUpload {
    const key = options.key as string;
    if (!key) throw new Error("options.key is required for blob upload");
    return new ResumableUpload(this, file, key, options, this.persistence);
  }

  /** Namespace-scoped queue (created lazily). */
  queue(options?: UploadQueueOptions): UploadQueue {
    if (!this._queue) {
      this._queue = new UploadQueue(this, this.ns, options);
    }
    return this._queue;
  }

  private async uploadDirect(file: File, key: string, options?: UploadOptions): Promise<Document<BlobMeta> | undefined> {
    const headers: Record<string, string> = {};
    const ct = options?.contentType || file.type;
    if (ct) headers["Content-Type"] = ct;

    const res = await this.client.dispatch<{ data: BlobMeta }>(
      "system:blobs:blob:upload",
      {
        arguments: { ns: this.ns, key },
        modifiers: options?.overwrite ? { overwrite: "true" } : undefined,
        payload: file,
        headers,
        bodyType: "blob",
      },
    );
    return asDoc(res.data!.data);
  }

  async beginUpload(props: {
    key: string;
    size: number;
    contentType?: string;
    blockSize?: number;
    overwrite?: boolean;
  }): Promise<UploadBeginResult> {
    const res = await this.client.dispatch<{ data: UploadBeginResult }>(
      "system:blobs:blob:begin",
      {
        arguments: { ns: this.ns },
        modifiers: props.overwrite ? { overwrite: "true" } : undefined,
        payload: {
          key: props.key,
          size: props.size,
          content_type: props.contentType,
          block_size: props.blockSize,
        },
      },
    );
    return res.data!.data;
  }

  async uploadChunk(props: {
    sessionId: string;
    offset: number;
    data: Blob;
    sha256?: string;
    signal?: AbortSignal;
  }): Promise<number> {
    const headers: Record<string, string> = {
      "X-Session-ID": props.sessionId,
      "X-Offset": String(props.offset),
    };
    if (props.sha256) headers["X-Chunk-SHA256"] = props.sha256;
    const res = await this.client.dispatch<{ data: { total: number } }>(
      "system:blobs:blob:chunk",
      { arguments: { ns: this.ns }, headers, payload: props.data, bodyType: "blob", signal: props.signal },
    );
    return res.data!.data.total;
  }

  async completeUpload(props: { sessionId: string; overwrite?: boolean }): Promise<Document<BlobMeta> | undefined> {
    const res = await this.client.dispatch<{ data: BlobMeta }>(
      "system:blobs:blob:complete",
      {
        arguments: { ns: this.ns },
        headers: { "X-Session-ID": props.sessionId },
        modifiers: props.overwrite ? { overwrite: "true" } : undefined,
      },
    );
    return asDoc(res.data!.data);
  }

  async progress(sessionId: string): Promise<UploadProgressResult> {
    const res = await this.client.dispatch<{ data: UploadProgressResult }>(
      "system:blobs:blob:progress",
      { arguments: { ns: this.ns }, modifiers: { session_id: sessionId } },
    );
    return res.data!.data;
  }

  async abort(sessionId: string): Promise<void> {
    await this.client.dispatch("system:blobs:blob:abort", {
      arguments: { ns: this.ns },
      headers: { "X-Session-ID": sessionId },
    });
  }

  async subscribe(_scope: string, _callback: (event: StoreEvent) => void): Promise<() => void> {
    throw new Error("Subscription not supported for blobs");
  }

  async notify(_event: StoreEvent): Promise<void> {
    throw new Error("Notify not supported for blobs");
  }

  stream(_options: Record<string, unknown>, _onStreamChange: () => void): {
    stream: () => AsyncIterable<Document<BlobMeta>>;
    cancel: () => void;
    status: () => "active" | "cancelled" | "completed";
  } {
    throw new Error("Stream not supported for blobs");
  }

  page(_options?: Record<string, unknown>): PagedData<BlobMeta> {
    return this.pager;
  }

  async download(key: string): Promise<{ data: Blob; contentType: string }> {
    const res = await this.client.dispatch<Blob>(
      "system:blobs:blob:download",
      { arguments: { ns: this.ns, key }, responseType: "blob" },
    );
    const blob = res.data!;
    return { data: blob, contentType: blob.type };
  }
}

export class HestiaBlobClient {
  private apiPrefix: string;

  constructor(
    private client: Transport,
    apiPrefix: string = "/api",
    private uploadPersistence?: UploadPersistence,
  ) {
    this.apiPrefix = apiPrefix;
  }

  async namespaces(): Promise<NamespaceInfo[]> {
    const res = await this.client.dispatch<{
      data: { namespaces: NamespaceInfo[] };
    }>("system:blobs:namespace:list");
    return res.data?.data?.namespaces ?? [];
  }

  async createNamespace(data: CreateNamespaceRequest): Promise<NamespaceInfo> {
    const res = await this.client.dispatch<{ data: NamespaceInfo }>(
      "system:blobs:namespace:create",
      { arguments: { ns: data.ns }, payload: { display_name: data.display_name } },
    );
    return res.data!.data;
  }

  async deleteNamespace(ns: string): Promise<void> {
    await this.client.dispatch("system:blobs:namespace:delete", {
      arguments: { ns },
    });
  }

  blob(namespace: string, key: string) {
    return this.client.routeUrl("system:blobs:blob:download", { ns: namespace, key });
  }

  namespace(ns: string): BlobNamespace {
    return new BlobNamespace(this.client, ns, this.uploadPersistence);
  }
}
