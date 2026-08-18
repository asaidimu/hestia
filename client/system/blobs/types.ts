import type { Document } from "../../core/types"

export interface NamespaceInfo {
  id: string
  display_name: string
}

export interface BlobMeta {
  key: string
  namespace_id: string
  content_type: string
  size: number
  created_at: string
  updated_at?: string
  custom?: Record<string, any>
}

export type BlobDocument = Document<BlobMeta>

export interface ListBlobsRequest {
  prefix?: string
  limit?: number
}

export interface CreateNamespaceRequest {
  display_name?: string
  ns: string
}

export interface UploadBeginResult {
  session_id: string
  key: string
  offset: number
  block_size: number
}

export interface UploadProgressResult {
  total: number
  ranges: { start: number; end: number }[]
  block_size: number
  expected_size: number
}

export type UploadPhase = "hash" | "upload"

export interface UploadProgressInfo {
  /** Current phase: hashing blocks or uploading chunks. */
  phase: UploadPhase
  /** Bytes pre-hashed during the hash phase. */
  hashed: number
  /** Total bytes to pre-hash (== total). */
  hashTotal: number
  /** Bytes persisted on the server. */
  uploaded: number
  /** Total file size in bytes. */
  total: number
  /** Overall progress in [0, 1]. */
  percent: number
  /** EWMA-smoothed upload throughput in bytes/second. */
  bytesPerSecond: number
  /** Estimated seconds remaining (0 when idle or done). */
  etaSeconds: number
  /** SHA-256 of the concatenated block hashes, set once hashing completes. */
  fingerprint?: string
}

export type UploadStatus =
  | "queued"
  | "hashing"
  | "uploading"
  | "paused"
  | "completing"
  | "completed"
  | "error"
  | "cancelled"

export interface UploadOptions {
  key?: string
  contentType?: string
  overwrite?: boolean
  /** Preferred block size in bytes; the server may override it. */
  blockSize?: number
  /** Files at or below this size use the direct single-shot upload. */
  threshold?: number
  /** Initial max in-flight chunk concurrency (default 4, AIMD-adjustable). */
  concurrency?: number
  /** Per-chunk retry count (default 3). */
  retries?: number
  onProgress?: (info: UploadProgressInfo) => void
  /** Aborts/cancels the upload (best-effort server session abort). */
  signal?: AbortSignal
  /** Force the resumable/staged protocol even for small files. */
  forceStaged?: boolean
}

/**
 * Serializable snapshot of an in-flight upload, used by the persistence
 * layer so an interrupted upload can be restored and resumed later. The
 * `file` reference is only preserved by structured-clone capable adapters
 * (e.g. IndexedDBPersistence); JSON-based adapters drop it, in which case
 * the caller must supply the same File again when resuming.
 */
export interface UploadJobSnapshot {
  id: string
  key: string
  name: string
  size: number
  type: string
  file?: File
  sessionId: string | null
  blockSize: number
  blockHashes: string[]
  fingerprint: string
  uploadedBytes: number
  status: UploadStatus
  error: string | null
}

export interface UploadJobsState {
  jobs: Record<string, UploadJobSnapshot>
}

/** Minimal persistence abstraction the upload engine writes snapshots to. */
export interface UploadPersistence {
  save(snapshot: UploadJobSnapshot): Promise<void> | void
  load(): Promise<UploadJobSnapshot[]> | UploadJobSnapshot[]
  remove(id: string): Promise<void> | void
}

export interface UploadQueueOptions {
  /** Maximum concurrently running uploads (default 3). */
  maxConcurrent?: number
  persistence?: UploadPersistence
}
