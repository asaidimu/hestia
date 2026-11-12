// @note #cruft-20260821-020 observation : Old-style handler functions in blobs/handler.go
// @status resolved
// @priority=P2
// @tags=#cruft,#note
// @see #8uuufn
// No action needed — dynamically registered via RegisterBlobHandlers, cannot use static codegen.
//
// This file contains NewListNamespacesHandler, NewCreateNamespaceHandler,
// NewDeleteNamespaceHandler, NewListBlobsHandler, NewHeadBlobHandler,
// NewUploadBlobHandler, NewDownloadBlobHandler, NewUpdateBlobHandler,
// NewDeleteBlobHandler, and other blob handlers — all using the old pattern
// of returning abstract.MessageHandler directly with manual input extraction.
//
// Unlike other packages, these handlers are NOT superseded by generated
// registrations. They are registered dynamically via RegisterBlobHandlers
// when a new namespace is created, which is a runtime operation that cannot
// be handled by static registrations. These are legitimate old-style handlers
// that serve a different purpose.
//
// Resolution: no action needed — these handlers are dynamically registered
// and cannot be replaced by generated registrations.
package blobs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/common"

	bserrors "github.com/asaidimu/blobs/errors"
	"github.com/asaidimu/blobs/staging"

	"github.com/asaidimu/hestia/core/abstract"
	blobutil "github.com/asaidimu/hestia/core/system/blobs/store"
	"github.com/asaidimu/hestia/core/system/blobs/model"
	"github.com/asaidimu/hestia/core/runtime"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

func NewListNamespacesHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		namespaces, err := svc.ListNamespaces(ctx)
		if err != nil {
			return nil, fmt.Errorf("list namespaces: %w", err)
		}

		views := make([]NamespaceView, len(namespaces))
		for i, ns := range namespaces {
			views[i] = NamespaceView{ID: ns.ID, DisplayName: ns.DisplayName, Public: ns.Public, Custom: ns.Custom}
		}

		return dispatch.NewDocumentResultFrom(&NamespaceListDocument{Namespaces: views})
	}
}

// @note #blob-20261029-010 todo : Namespace creation is not atomic and its id is barely validated
// @priority medium
// @tags #blobs #robustness
// @author claude
// @see #blob-20261029-011
//
// The steps are svc.CreateNamespace, SeedNamespaceBindings, RegisterBlobHandlers, with no rollback. A failure after the first leaves a namespace that exists but has no bindings or handlers, and retrying create then fails because it already exists, so it needs manual cleanup.
// The id falls back to display_name when ns is empty and the only check is non-empty: spaces, ':' and '/' are accepted, and the ids "blob" or "namespace" produce per-namespace operation names (system:blobs:blob:*, system:blobs:namespace:list|stats|verify|delete) that collide with the static generic routes. Whether the blobs library itself rejects such ids was not verified.
// Fix: validate ids (lowercase slug plus reserved words), drop the display_name fallback, and compensate (delete the namespace) on later failure. Deletion has the mirror problem: #blob-20261029-011.

func NewCreateNamespaceHandler(svc blobutil.BlobStore, mgr *staging.Manager, policyOp abstract.BindingPolicyStore, registry abstract.Registry) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input NsCreateInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		nsID := input.NS
		if nsID == "" {
			nsID = input.DisplayName
		}
		if nsID == "" {
			return nil, common.NewSystemError("VALIDATION_ERROR", "namespace ID is required")
		}

		var opts []blobutil.NamespaceOption
		if input.Public {
			opts = append(opts, blobutil.WithPublic(true))
		}
		if len(input.Custom) > 0 {
			opts = append(opts, blobutil.WithCustom(input.Custom))
		}

		if err := svc.CreateNamespace(ctx, nsID, input.DisplayName, opts...); err != nil {
			return nil, fmt.Errorf("create namespace: %w", err)
		}

		if err := SeedNamespaceBindings(ctx, policyOp, nsID); err != nil {
			return nil, fmt.Errorf("seed namespace operations: %w", err)
		}

		if err := RegisterBlobHandlers(registry, svc, mgr, nsID); err != nil {
			return nil, fmt.Errorf("register blob handlers: %w", err)
		}

		return dispatch.NewDocumentResultFrom(&NamespaceView{ID: nsID, DisplayName: input.DisplayName, Public: input.Public, Custom: input.Custom})
	}
}

// @note #blob-20261029-011 todo : Namespace deletion destroys blobs without confirmation and can half-complete
// @priority medium
// @tags #blobs #robustness
// @author claude
// @see #blob-20261029-010
//
// Order is UnregisterBlobHandlers, delete each binding (returning on the first failure), ReloadPolicies, then svc.DeleteNamespace. A failure after the first step leaves a namespace that still holds data but has no handlers or bindings, and a namespace containing blobs is deleted outright: no emptiness check, no force or confirm flag, no soft delete.
// Proposal: make each step idempotent so a retry converges, and require force=true for a non-empty namespace. Creation has the mirror problem: #blob-20261029-010.

func NewDeleteNamespaceHandler(svc blobutil.BlobStore, policyOp abstract.BindingPolicyStore, registry abstract.Registry) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input NsInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		nsID := input.NS
		if nsID == "" {
			return nil, common.NewSystemError("VALIDATION_ERROR", "namespace ID is required")
		}

		UnregisterBlobHandlers(registry, nsID)

		for _, op := range blobOps {
			opName := "system:blobs:" + nsID + ":" + op.Suffix
			if err := policyOp.DeleteBinding(ctx, opName); err != nil {
				return nil, fmt.Errorf("delete operation %s: %w", opName, err)
			}
		}

		if err := policyOp.ReloadPolicies(ctx); err != nil {
			return nil, fmt.Errorf("reload policies: %w", err)
		}

		if err := svc.DeleteNamespace(ctx, nsID); err != nil {
			return nil, fmt.Errorf("delete namespace: %w", err)
		}

		return &abstract.Result{}, nil
	}
}

// @note #blob-20261029-007 todo : List has no cursor, delimiter or truncation signal, and limit 0 returns everything
// @priority medium
// @tags #blobs #api #ergonomics
// @author claude
// @see #blob-20261029-021
//
// BlobListInput is {prefix, limit}; the store's List takes the same two values and the response is {blobs: [...]}. A namespace with many blobs comes back in one body, ordering is not specified here, and a caller who sets limit cannot tell whether more exist or fetch the next page. There is no delimiter either, so folder-style listings (keys containing '/') must download every key.
// Proposal: opaque cursor in the request, next_cursor/has_more in BlobListDocument, an optional delimiter returning common_prefixes, and a server-side maximum limit. Client side of the same gap: #blob-20261029-021.

func NewListBlobsHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input BlobListInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		if input.NS == "" {
			return nil, common.NewSystemError("VALIDATION_ERROR", "namespace ID is required")
		}

		blobs, err := svc.Namespace(input.NS).List(ctx, input.Prefix, input.Limit)
		if err != nil {
			return nil, fmt.Errorf("list blobs: %w", err)
		}

		items := make([]BlobMetaView, len(blobs))
		for i := range blobs {
			items[i] = blobMetaView(&blobs[i])
		}

		return dispatch.NewDocumentResultFrom(&BlobListDocument{Blobs: items})
	}
}

func NewHeadBlobHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input BlobKeyInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		meta, err := svc.Namespace(input.NS).Head(ctx, input.Key)
		if err != nil {
			return nil, mapBlobError(err)
		}

		view := blobMetaView(meta)
		return dispatch.NewDocumentResultFrom(&view)
	}
}

// @note #blob-20261029-005 issue : Empty (zero-byte) blobs cannot be stored
// @priority medium
// @tags #blobs #ergonomics
// @author claude
// @see #blob-20261029-018
//
// NewUploadBlobHandler rejects len(payload) == 0 ("request body is required") and NewBeginUploadHandler rejects size <= 0, so neither path accepts an empty file (placeholders, .gitkeep, empty exports). The TS client sends every file at or below its 16 MiB threshold, including 0 bytes, down the direct path, so the caller just receives VALIDATION_ERROR.
// Fix: accept empty bodies on the direct path (a missing body and an empty one are different things), or document the limit and fail early in the client. Related client behaviour: #blob-20261029-018.

func NewUploadBlobHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input BlobUploadInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		payload := append([]byte(nil), input.Payload...)
		msg.Input().Release()

		if len(payload) == 0 {
			return nil, common.NewSystemError("VALIDATION_ERROR", "request body is required")
		}
		if len(payload) > maxDirectUploadBytes {
			return nil, common.NewSystemError("VALIDATION_ERROR",
				fmt.Sprintf("direct upload exceeds %d byte limit; use the resumable upload protocol", maxDirectUploadBytes))
		}

		if err := ensureKeyWritable(ctx, svc, input.NS, input.Key, input.Overwrite == "true"); err != nil {
			return nil, err
		}

		meta, err := svc.Namespace(input.NS).Put(ctx, input.Key, input.ContentType, bytes.NewReader(payload))
		if err != nil {
			return nil, mapBlobError(err)
		}

		view := blobMetaView(meta)
		return dispatch.NewDocumentResultFrom(&view)
	}
}

// @note #blob-20261029-004 todo : Download buffers the whole blob; upload and chunk handlers copy their payloads
// @priority medium
// @tags #blobs #performance #ergonomics
// @author claude
// @see #blob-20261029-003
//
// NewDownloadBlobHandler does Head then Get and io.ReadAll into abstract.Blob.Data, so memory per request equals the blob size, with no Range and no streaming. Resumable uploads can create blobs far beyond maxDirectUploadBytes (16 MiB), so a file that uploads fine can be impractical to read back. Head+Get is also two lookups, so the content type can describe a replaced body.
// Related copies: NewUploadBlobHandler and NewUploadChunkHandler do append([]byte(nil), input.Payload...) before Release(), and a chunk may be up to 256 MiB (maxUploadChunkBytes), times the client's chunk concurrency (up to 6). Every system:blobs:* route also gets the large body limit (register.go, S-8), including JSON-only ones (list, update, begin, rename, namespace:create) that only need the small default.
// serializeResponse answers a BlobChannel result with an empty 200, so a streaming blob result exists in the model but is not wired through HTTP. Proposal: stream downloads, honour Range, and apply the large limit only to upload and chunk. Header gaps are in #blob-20261029-003.

func NewDownloadBlobHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input BlobKeyInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		meta, err := svc.Namespace(input.NS).Head(ctx, input.Key)
		if err != nil {
			return nil, mapBlobError(err)
		}

		rc, err := svc.Namespace(input.NS).Get(ctx, input.Key)
		if err != nil {
			return nil, mapBlobError(err)
		}
		defer rc.Close()

		data, err := io.ReadAll(rc)
		if err != nil {
			return nil, fmt.Errorf("read blob: %w", err)
		}

		return &abstract.Result{
			Blob: abstract.Blob{
				Data:        data,
				ContentType: meta.ContentType,
			},
		}, nil
	}
}

func NewUpdateBlobHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input BlobUpdateInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		meta, err := svc.Namespace(input.NS).UpdateMetadata(ctx, input.Key, input.Custom)
		if err != nil {
			return nil, mapBlobError(err)
		}

		view := blobMetaView(meta)
		return dispatch.NewDocumentResultFrom(&view)
	}
}

func NewDeleteBlobHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input BlobKeyInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		if _, err := svc.Namespace(input.NS).Head(ctx, input.Key); err != nil {
			return nil, mapBlobError(err)
		}

		if err := svc.Namespace(input.NS).Delete(ctx, input.Key); err != nil {
			return nil, mapBlobError(err)
		}

		return &abstract.Result{}, nil
	}
}

func NewRenameBlobHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input BlobRenameInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		if err := svc.Namespace(input.NS).Rename(ctx, input.Key, input.NewKey); err != nil {
			return nil, mapBlobError(err)
		}

		view := MessageOutput{Message: "renamed"}
		return dispatch.NewDocumentResultFrom(&view)
	}
}

func NewStatsNamespaceHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input NsInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		stats, err := svc.Namespace(input.NS).Stats(ctx)
		if err != nil {
			return nil, mapBlobError(err)
		}

		view := NamespaceStatsDocument{
			NamespaceID:  stats.NamespaceID,
			BlobCount:    stats.BlobCount,
			BytesStored:  stats.BytesStored,
			BytesPhysical: stats.BytesPhysical,
			ChunkCount:   stats.ChunkCount,
			DeadBytes:    stats.DeadBytes,
			DeadChunks:   stats.DeadChunks,
			SegmentCount: stats.SegmentCount,
		}
		return dispatch.NewDocumentResultFrom(&view)
	}
}

func NewVerifyNamespaceHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input NsInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		if err := svc.Namespace(input.NS).Verify(ctx); err != nil {
			return nil, mapBlobError(err)
		}

		view := MessageOutput{Message: "ok"}
		return dispatch.NewDocumentResultFrom(&view)
	}
}

func NewCompactNamespaceHandler(svc blobutil.BlobStore) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input NsInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		msg.Input().Release()

		result, err := svc.Namespace(input.NS).Compact(ctx)
		if err != nil {
			return nil, mapBlobError(err)
		}

		view := model.CompactResultDocument{
			BlobsRemoved:      result.BlobsRemoved,
			ChunksRemoved:     result.ChunksRemoved,
			BytesFreed:        result.BytesFreed,
			SegmentsCompacted: result.SegmentsCompacted,
		}
		return dispatch.NewDocumentResultFrom(&view)
	}
}

// @note #blob-20261029-006 issue : overwrite=false is check-then-put, so concurrent uploads to one key silently overwrite each other
// @priority medium
// @tags #blobs #correctness
// @author claude
// @see #blob-20261029-002
//
// ensureKeyWritable does Head, and the caller then does Put/PutCustom. Two requests that both see NOT_FOUND both succeed, the later one wins and neither receives ALREADY_EXISTS. The same applies at begin and again at complete (the repeated check narrows nothing). The contract "overwrite=false never replaces an existing blob" is therefore advisory under concurrency. Not exercised.
// Fix: an atomic put-if-absent (or a per-key lock) in the store / blobs library. Complete-time behaviour is in #blob-20261029-002.

// ensureKeyWritable enforces overwrite protection: unless overwrite is true,
// an existing key rejects the write with runtime.ErrAlreadyExists.
func ensureKeyWritable(ctx context.Context, svc blobutil.BlobStore, nsID, key string, overwrite bool) error {
	if key == "" {
		return common.NewSystemError("VALIDATION_ERROR", "blob key is required")
	}
	if overwrite {
		return nil
	}
	_, err := svc.Namespace(nsID).Head(ctx, key)
	if err == nil {
		return runtime.ErrAlreadyExists.WithCause(fmt.Errorf("blob key %q already exists", key))
	}
	var notFound *bserrors.NotFoundError
	if errors.As(err, &notFound) {
		return nil
	}
	return mapBlobError(err)
}

// blobMetaView maps store metadata into its typed wire shape, rendering
// timestamps as RFC3339 strings and omitting an untouched UpdatedAt.
func blobMetaView(m *blobutil.BlobMeta) BlobMetaView {
	view := BlobMetaView{
		Key:         m.Key,
		NamespaceID: m.NamespaceID,
		ContentType: m.ContentType,
		Size:        m.Size,
		CreatedAt:   m.CreatedAt.Format(time.RFC3339Nano),
		Custom:      m.Custom,
	}
	if !m.UpdatedAt.IsZero() {
		updated := m.UpdatedAt.Format(time.RFC3339Nano)
		view.UpdatedAt = &updated
	}
	return view
}

// @note #blob-20261029-012 todo : Inconsistent error model across blob operations
// @priority medium
// @tags #blobs #errors #ergonomics
// @author claude
// @see #blob-20261029-025
//
// mapBlobError covers head, upload, download, update, delete, rename, stats, verify and compact, but list, namespace create/delete and begin wrap with fmt.Errorf (untyped, surfaced as an internal error), and the NamespacedDispatcher hydrator does the same for an unknown namespace. A mistyped namespace is therefore not NOT_FOUND, and the client's read(), which returns undefined only on code NOT_FOUND, throws instead.
// Similar failures use different codes: VALIDATION_ERROR for bad parameters, INVALID_REQUEST for staging size/checksum/alignment problems. Some responses are ad hoc: rename answers {message: "renamed"} and verify {message: "ok"} with no declared output schema, so clients discard them, and verify offers no per-blob report (a CORRUPTION error is the only detail).
// Fix: one mapper for blobs, staging and the hydrator, a short code table in docs/guide/blobs.md, and structured outputs (rename returns the BlobMeta, verify returns a report). Client side: #blob-20261029-025.

func mapBlobError(err error) error {
	var notFound *bserrors.NotFoundError
	if errors.As(err, &notFound) {
		return runtime.ErrNotFound.WithCause(err)
	}
	var exists *bserrors.AlreadyExistsError
	if errors.As(err, &exists) {
		return runtime.ErrAlreadyExists.WithCause(err)
	}
	var corruption *bserrors.CorruptionError
	if errors.As(err, &corruption) {
		return common.NewSystemError("CORRUPTION", corruption.Error()).WithCause(err)
	}
	return fmt.Errorf("blob: %w", err)
}

// @note #blob-20261029-013 issue : blobOps, the static registrations and the dispatcher rewrite disagree
// @priority low
// @tags #blobs #cruft
// @author claude
//
// blobOps seeds a binding for "admin" but RegisterBlobHandlers registers no admin handler, so that binding is dead. "compact" has no per-namespace op at all. The NamespacedDispatcher rewrites only the system:blobs:blob: prefix, so system:blobs:namespace:stats|verify|compact (static, rule=root) appear never to consult the per-namespace stats/verify bindings that SeedNamespaceBindings creates; a non-root user granted <ns>:stats could not use the client's stats() (from reading, not exercised). compact is also registered with intent=create although it is a maintenance operation, and every binding is seeded as "root", so each grant is a manual policy edit.
// Fix: make one op table the source of truth for registrations, bindings and rewrites.

type BlobOp struct {
	Suffix, RuleKey, Desc string
}

var blobOps = []BlobOp{
	{"list", "root", "List blobs"},
	{"head", "root", "Get blob metadata"},
	{"upload", "root", "Upload a blob"},
	{"download", "root", "Download a blob"},
	{"delete", "root", "Delete a blob"},
	{"update", "root", "Update blob metadata"},
	{"rename", "root", "Rename a blob"},
	{"stats", "root", "Get namespace stats"},
	{"verify", "root", "Verify namespace integrity"},
	{"begin", "root", "Begin a resumable blob upload"},
	{"chunk", "root", "Upload a chunk of a resumable blob upload"},
	{"complete", "root", "Complete a resumable blob upload"},
	{"progress", "root", "Report progress of a resumable blob upload"},
	{"abort", "root", "Abort a resumable blob upload"},
	{"admin", "root", "Administer blob namespace"},
}

func BlobOps() []BlobOp { return blobOps }

func SeedNamespaceBindings(ctx context.Context, policyOp abstract.BindingPolicyStore, nsID string) error {
	for _, op := range blobOps {
		opName := "system:blobs:" + nsID + ":" + op.Suffix
		if err := policyOp.EnsureBinding(ctx, opName, op.RuleKey); err != nil {
			return fmt.Errorf("register operation %s: %w", opName, err)
		}
	}
	return policyOp.ReloadPolicies(ctx)
}

func RegisterBlobHandlers(registry abstract.Registry, svc blobutil.BlobStore, mgr *staging.Manager, nsID string) error {
	entries := []struct {
		suffix  string
		handler abstract.MessageHandler
	}{
		{"list", NewListBlobsHandler(svc)},
		{"head", NewHeadBlobHandler(svc)},
		{"upload", NewUploadBlobHandler(svc)},
		{"download", NewDownloadBlobHandler(svc)},
		{"delete", NewDeleteBlobHandler(svc)},
		{"update", NewUpdateBlobHandler(svc)},
		{"rename", NewRenameBlobHandler(svc)},
		{"stats", NewStatsNamespaceHandler(svc)},
		{"verify", NewVerifyNamespaceHandler(svc)},
		{"begin", NewBeginUploadHandler(svc, mgr)},
		{"chunk", NewUploadChunkHandler(mgr)},
		{"complete", NewCompleteUploadHandler(svc, mgr)},
		{"progress", NewProgressUploadHandler(mgr)},
		{"abort", NewAbortUploadHandler(mgr)},
	}
	for _, e := range entries {
		name := "system:blobs:" + nsID + ":" + e.suffix
		if err := registry.RegisterHandler(name, e.handler, abstract.HandlerInfo{
			Name:        name,
			Description: fmt.Sprintf("%s in namespace %q", blobOpDesc(e.suffix), nsID),
			Enabled:     true,
		}); err != nil {
			return fmt.Errorf("register %s: %w", name, err)
		}
	}
	return nil
}

func UnregisterBlobHandlers(registry abstract.Registry, nsID string) {
	for _, op := range blobOps {
		registry.DeleteHandler("system:blobs:" + nsID + ":" + op.Suffix)
	}
}

func blobOpDesc(suffix string) string {
	for _, op := range blobOps {
		if op.Suffix == suffix {
			return op.Desc
		}
	}
	return suffix
}
