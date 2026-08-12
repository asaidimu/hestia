# Blob Refactor Plan

Align hestia's blob integration with the reference examples in `~/projects/blobs/examples`
(`basic`, `movies`, `staging`). Current gaps identified:

1. **Uploads aren't resumable** — single-shot whole-body `Put` buffers the entire file in
   memory (twice, via the base64 JSON envelope) and offers no resume/progress.
2. **`Content-Type` header is silently dropped** — the input meta-schema only allows root
   fields `arguments/modifiers/payload/_id_/_metadata_`, so the `content_type` injection in
   `core/interface/http/doc.go` is dead code and the store always sniffs.
3. **Fasthttp 4MB body cap** — `transport_fasthttp.go` `Start()` never sets
   `MaxRequestBodySize`, so any chunk/upload >4MB is rejected.
4. **`Put` silently overwrites existing keys** — `store.Store` Put re-points an existing ref;
   both examples guard with `uniqueStoreKey`-style helpers.
5. **Lifecycle leak** — `BlobSvc.Close()` / reaper stop are never called on shutdown.

Decisions (confirmed):
- **Upload scope:** full resumable client (pre-hashing, resume via `progress()`, retries, progress callbacks).
- **Download streaming:** deferred — `io.ReadAll` download left as-is.
- **Overwrite semantics:** reject by default, `?overwrite=true` opt-in.

## Phase 1 — Server: staging protocol (blobs `staging`, already in go.mod v1.3.1)

### New handlers (`core/feature/blobs`)

- `system:blobs:blob:begin` — JSON payload (`filename`, `size`, `contentType`, `blockSize?`)
  → `staging.Begin` → `{session_id, key, block_size}`. Reject if key exists unless `?overwrite=true`.
- `system:blobs:blob:chunk` — `FieldTypeBytes` payload (raw chunk), headers
  `X-Session-ID`/`X-Offset`/`X-Chunk-SHA256` → `staging.WriteChunk` → `{total}`. Cap chunk size
  (256MiB, matches `examples/movies/handlers_upload.go`).
- `system:blobs:blob:complete` — `staging.Complete` → `ns.Put(ctx, key, cu, store.PutOptions{ContentType, Custom})`
  (single-pass stream) → `cu.Finalize()` → blob meta.
- `system:blobs:blob:progress` — `staging.Ranges`/`BlockSize`/`ExpectedSize` → missing ranges for resume.
- `system:blobs:blob:abort` — `staging.Abort`.

Register in `feature.go` (`Registrations`), add ops/bindings in `handler.go` (`blobOps`,
`SeedNamespaceBindings`, `RegisterBlobHandlers`, `UnregisterBlobHandlers`), extend mocks/tests.

### Wiring

- Add `staging.Manager` to the blob service; `StagingDir` config (default `BlobsDir/staging`);
  start `StartReaper(5m, 6h)`.
- **Header plumbing (required):** add `runtime.Input.HeaderFields map[string]string`;
  generalize `BuildInputDocument` to inject them under a new `headers` root object; widen the
  input meta-schema enum (`core/runtime/dispatch/input.go:41`) with `"headers"`. DTO structs
  declare fields like `input:"headers.session_id"`; move the upload's `content_type` to
  `headers.content_type` (fixes the dropped Content-Type); update handler + unit tests.
- **Server bug fix:** set `MaxRequestBodySize` (10GiB, per `examples/staging/main.go`) in
  `transport_fasthttp.go` `Start()`.
- **Lifecycle:** add a module `Stop` hook that stops the reaper and calls `BlobSvc.Close()`.
- Regenerate `client/core/routes.gen.ts` via `go run ./cmd/gen-routes`.

## Phase 2 — Client: full resumable uploader (`client/blobs/store.ts`)

- `BlobNamespace` gains `beginUpload`, `uploadChunk`, `completeUpload`, `progress`, `abort`
  (via `client.dispatch` with `headers` + `bodyType: "blob"`), plus a resumable `upload` flow:
  - Slice the `File` into blocks of `blockSize` (from `begin`, default 8MiB), pre-hash each
    with `crypto.subtle` SHA-256.
  - Upload blocks with `X-Offset` + `X-Chunk-SHA256`, bounded in-flight concurrency (AIMD-style),
    retries with backoff, abort/pause support.
  - On resume, call `progress()` → upload only missing ranges.
  - `onProgress` callback for UI progress/ETA.
- Keep direct `upload()` for small files (auto-switch to staged above a threshold, e.g. 16MiB).
  `download()`/`blob()` URL unchanged.
- Update `client/blobs/blobs.test.ts`.

## Phase 3 — Overwrite protection

- Direct upload and staging `begin`: if key exists and `?overwrite=true` is not set →
  `runtime.ErrAlreadyExists` (mapped to 409).
- Re-check at `complete` (TOCTOU-safe), also honoring `overwrite`.

## Verification

- Go: handler unit tests (mock store + temp staging dir) for
  begin/chunk/complete/progress/abort/reject-on-collision; transport/header-injection test;
  `go vet` / `go build ./...` and existing `./core/...` tests.
- Client: unit tests for chunking/resume logic against a mock transport.
- Manual: exercise against the auto-reloading test server on :8070 (session via
  `POST /api/system/session`, then a staged upload + interrupted-chunk resume).

## Risks / notes

- Chunk payloads still round-trip through the base64 JSON envelope on the server
  (`FieldTypeBytes`), so a chunk is buffered ~2× in RAM — acceptable at 8MiB blocks;
  true `RequestBodyStream` streaming would be a later optimization.
- Fasthttp has no `ServeContent`; Range streaming is deliberately deferred.
