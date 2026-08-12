# Blob Upload Engine — Full Resumable Upload Port

Port every feature of the prototype client in `~/projects/blobs/examples/staging/main.go`
(the `TransferEngine` inline HTML) into the hestia TypeScript SDK, plus the streaming
pre-hash memory fix. Prototype features: streaming pre-hash, fingerprint, AIMD
concurrency, EWMA throughput/ETA, per-chunk timeout + backoff retries, worker pool,
missing-range resume, full state machine (pause/resume/cancel/retry), job queue,
and persistence via the existing `@asaidimu/utils-persistence` abstraction.

Decision log (confirmed):
- **Network profile:** adaptive from upload (no `/ping`/`/probe` — hestia has none).
  Initial blockSize from `begin` (8MiB), concurrency 4; adapt via AIMD/EWMA.
- **Queue:** implemented in the SDK (`UploadQueue`, max 3 concurrent), per-namespace.
- **Persistence:** use `@asaidimu/utils-persistence` `SimplePersistence<T>`; default
  `EphemeralPersistence`; callers may pass `IndexedDBPersistence`/`WebStoragePersistence`.
- **UI (DOM dashboard):** NOT in scope — engine + queue only; app-layer.
- `upload()` keeps returning `Promise<Document|undefined>`; new `createUpload()` and
  `queue()` expose the full engine.

---

## Phase 1 — Types (`client/blobs/types.ts`)

- [x] Add `UploadStatus` union: `"queued" | "hashing" | "uploading" | "paused" |
      "completing" | "completed" | "error" | "cancelled"`.
- [x] Extend `UploadProgressInfo` with `phase: "hash" | "upload"`, `hashed`,
      `hashTotal`, and `fingerprint?: string`.
- [x] Add `UploadOptions` fields: `concurrency`, `retries`, `onProgress`, `signal`,
      `forceStaged`, `threshold`, `blockSize`, `contentType`, `overwrite`, `key`.
- [x] Add `UploadJobSnapshot`: `id`, `name`, `size`, `type`, `file?`, `sessionId`,
      `blockSize`, `blockHashes`, `fingerprint`, `uploadedBytes`, `status`, `error`.
- [x] Add `UploadJobsState` = `{ jobs: Record<string, UploadJobSnapshot> }` and a
      `UploadPersistence` adapter shape over `SimplePersistence<UploadJobsState>`.

## Phase 2 — Engine core (`client/blobs/store.ts`)

### Streaming pre-hash (memory fix)
- [x] Add `sliceFile(file, start, end): Blob` helper (Bun types lack `File.slice`;
      runtime has it — verified).
- [x] Remove whole-file `new Uint8Array(await file.arrayBuffer())` from `uploadStaged`.
- [x] Implement `hashBlocks()`: iterate blocks via `sliceFile().arrayBuffer()` →
      `sha256Hex`, one block in RAM at a time; cancellable via signal; yield every
      ~12ms; report `hashed`/`hashTotal` + phase `"hash"`.
- [x] Compute `fingerprint` = SHA-256 of concatenated block-hash hex strings.

### Concurrency & throughput
- [x] Implement `AimdController`: `consecutiveSuccesses`, +1 after 3 consecutive
      successes (max 6), halve on failure (min 1) gated by cooldown, `estThroughput()`.
- [x] Implement EWMA throughput (α=0.3) + rolling speed-sample window (last 5) +
      ETA computation in progress reporting.
- [x] Remove the old `runBounded` helper and `UploadBlock` interface once replaced.

### Upload pool
- [x] Implement `getMissingRanges(ranges, fileSize)` — gaps between server ranges.
- [x] Implement chunk cursor over missing ranges, slicing by `blockSize`.
- [x] Implement worker pool: spawn up to `maxConcurrency` workers, replacement spawn
      when a slot frees and cursor has more; pause/cancel checks.
- [x] Implement `uploadOneChunk`: body = `sliceFile()` Blob; headers
      `X-Session-ID`/`X-Offset`/`X-Chunk-SHA256`; per-chunk `AbortController` +
      timeout `max(6000, bytes/estBps*3)`; retries with backoff `1500*2^(attempt-1)`
      (max 3); success → AIMD `recordSuccess`; failure → `recordFailure`.

## Phase 3 — `ResumableUpload` controller

- [x] Class with fields: `status`, `blockSize`, `sessionId`, `uploaded`, `hashed`,
      `hashTotal`, `total`, `bytesPerSecond`, `etaSeconds`, `fingerprint`, `error`,
      `done: Promise<Document<BlobMeta>|undefined>`.
- [x] `run()` orchestrator: progress() → missingRanges → pool, loop until
      `uploaded >= size`, then complete() → "completed".
- [x] State transitions: queued/hashing/uploading/paused/completing/completed/error/cancelled.
- [x] `pause()` — set pause flag, abort in-flight chunk controllers, keep session.
- [x] `resume()` — re-run orchestrator loop from saved state.
- [x] `cancel()` — abort in-flight, call server `abort(sessionId)`, status cancelled.
- [x] `retry()` — from error, re-run loop.
- [x] `onProgress` fires with both hash and upload phases incl. `fingerprint`.
- [x] Persistence hooks: save/clear `UploadJobSnapshot` at begin, hash-done, pause,
      error, and completed (via `UploadJobStore`).

## Phase 4 — Queue (`UploadQueue`)

- [x] Class per namespace: `enqueue(file, options)` → `ResumableUpload` ("queued").
- [x] `pump()` scheduling with `maxConcurrent` (default 3).
- [x] `pauseAll()`, `resumeAll()`, `clearCompleted()`, `retry()`.
- [x] `restore()` — load persisted snapshots, mark "paused", ready to resume.
- [x] `summary()` — active/paused/done/failed/cancelled counts, total/uploaded bytes,
      aggregate speed + ETA.
- [x] Expose `BlobNamespace.createUploadQueue()` / `queue()`.

## Phase 5 — Wiring & compatibility

- [x] `BlobNamespace.upload()` stays backward-compatible (direct path for small
      files unchanged; staged path runs the engine internally).
- [x] Add `BlobNamespace.createUpload(file, options): ResumableUpload`.
- [x] Add `UploadJobStore` wrapper around `SimplePersistence<UploadJobsState>`
      (implements the engine's `UploadPersistence`; memory-only when omitted).
- [x] Thread optional `uploadPersistence` through `HestiaConfig` → `HestiaBlobClient`
      → `BlobNamespace`.
- [x] Default persistence adapter: no-op memory-only when none provided.
- [x] `beginUpload/uploadChunk/completeUpload/progress/abort` transport methods
      unchanged.

## Phase 6 — Tests (`client/blobs/blobs.test.ts`)

- [x] Streaming pre-hash: fingerprint correct; hash→upload phase progress transitions.
- [x] Resume via missing ranges: only gaps uploaded.
- [x] AIMD: success streak grows concurrency; failure halves it (cooldown-gated).
- [x] Per-chunk retry + backoff recovery; failed completion retried via `retry()`.
- [x] Pause/resume/cancel/retry state-machine transitions (incl. abort signal).
- [x] Queue: max-concurrency scheduling, pauseAll/resumeAll/clearCompleted, summary.
- [x] Persistence: save/load/clear round-trip with a mock `SimplePersistence`;
      restore → paused → resume.
- [x] Existing direct-upload tests stay green.

## Verification

- [x] `make test-server` then `cd client && bunx vitest --run` — all green.
- [x] `cd client && bunx tsc --noEmit -p tsconfig.json` — no new `blobs/*` errors
      (repo has pre-existing errors in auth/operations/policies).
- [x] `go test ./...` — unaffected (no server changes).
- [x] Manual sanity against auto-reloading test server on :8070 (session via
      `POST /api/system/session`, staged upload + interrupted-chunk resume).
