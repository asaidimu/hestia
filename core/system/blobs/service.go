package blobs

import (
	"context"

	"github.com/asaidimu/blobs/staging"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	blobutil "github.com/asaidimu/hestia/core/system/blobs/store"
	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/system/blobs/model"
)

// BlobsService owns the static blob registrations: namespace lifecycle and the
// generic system:blobs:blob:* operations. Per-namespace dynamic handlers
// (system:blobs:<ns>:*) stay runtime-registered by registerExistingBlobHandlers.
type BlobsService struct {
	blobStore blobutil.BlobStore
	staging   *staging.Manager
	policy    abstract.BindingPolicyStore
	registry  abstract.Registry
	logger    *zap.Logger
}

func NewBlobsService(rt abstract.Container) (*BlobsService, error) {
	storeSvc := abstract.MustResolve[*blobutil.Service](rt)
	logger := abstract.MustResolve[*zap.Logger](rt)
	policy := abstract.MustResolve[abstract.BindingPolicyStore](rt)
	registry := abstract.MustResolve[*runtime.LocalDispatcher](rt)
	return &BlobsService{
		blobStore: storeSvc,
		staging:   storeSvc.Staging(),
		policy:    policy,
		registry:  registry,
		logger:    logger,
	}, nil
}

// ListNamespaces lists all blob namespaces.
//
// @hestia.register(
//   name="system:blobs:namespace:list",
//   intent="query",
//   rule="administrator",
//   description="List blob namespaces",
//   output="model.NamespaceListOutput",
// )
func (s *BlobsService) ListNamespaces(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
	return NewListNamespacesHandler(s.blobStore)(ctx, msg)
}

// CreateNamespace creates a blob namespace and seeds its per-namespace
// operations and handlers.
//
// @hestia.register(
//   name="system:blobs:namespace:create",
//   intent="create",
//   rule="administrator",
//   description="Create a blob namespace",
//   output="model.NamespaceOutput",
// )
func (s *BlobsService) CreateNamespace(ctx context.Context, msg abstract.Message, input *model.NsCreateInput) (*abstract.Result, error) {
	return NewCreateNamespaceHandler(s.blobStore, s.staging, s.policy, s.registry)(ctx, msg)
}

// DeleteNamespace deletes a blob namespace and its bindings.
//
// @hestia.register(
//   name="system:blobs:namespace:delete",
//   intent="delete",
//   rule="administrator",
//   resource_id="ns",
//   description="Delete a blob namespace",
// )
func (s *BlobsService) DeleteNamespace(ctx context.Context, msg abstract.Message, input *model.NsInput) (*abstract.Result, error) {
	return NewDeleteNamespaceHandler(s.blobStore, s.policy, s.registry)(ctx, msg)
}

// ListBlobs lists blobs in a namespace.
//
// @hestia.register(
//   name="system:blobs:blob:list",
//   intent="query",
//   rule="administrator",
//   description="List blobs in a namespace",
//   output="model.BlobListOutput",
// )
func (s *BlobsService) ListBlobs(ctx context.Context, msg abstract.Message, input *model.BlobListInput) (*abstract.Result, error) {
	return NewListBlobsHandler(s.blobStore)(ctx, msg)
}

// HeadBlob gets blob metadata.
//
// @hestia.register(
//   name="system:blobs:blob:head",
//   intent="query",
//   rule="administrator",
//   resource_id="key",
//   description="Get blob metadata",
//   output="model.BlobMetaOutput",
// )
func (s *BlobsService) HeadBlob(ctx context.Context, msg abstract.Message, input *model.BlobKeyInput) (*abstract.Result, error) {
	return NewHeadBlobHandler(s.blobStore)(ctx, msg)
}

// UploadBlob uploads a blob.
//
// @hestia.register(
//   name="system:blobs:blob:upload",
//   intent="create",
//   rule="administrator",
//   resource_id="key",
//   description="Upload a blob",
//   output="model.BlobMetaOutput",
// )
func (s *BlobsService) UploadBlob(ctx context.Context, msg abstract.Message, input *model.BlobUploadInput) (*abstract.Result, error) {
	return NewUploadBlobHandler(s.blobStore)(ctx, msg)
}

// DownloadBlob downloads a blob.
//
// @hestia.register(
//   name="system:blobs:blob:download",
//   intent="read",
//   rule="administrator",
//   resource_id="key",
//   description="Download a blob",
// )
func (s *BlobsService) DownloadBlob(ctx context.Context, msg abstract.Message, input *model.BlobKeyInput) (*abstract.Result, error) {
	return NewDownloadBlobHandler(s.blobStore)(ctx, msg)
}

// DeleteBlob deletes a blob.
//
// @hestia.register(
//   name="system:blobs:blob:delete",
//   intent="delete",
//   rule="administrator",
//   resource_id="key",
//   description="Delete a blob",
// )
func (s *BlobsService) DeleteBlob(ctx context.Context, msg abstract.Message, input *model.BlobKeyInput) (*abstract.Result, error) {
	return NewDeleteBlobHandler(s.blobStore)(ctx, msg)
}

// UpdateBlob updates blob metadata.
//
// @hestia.register(
//   name="system:blobs:blob:update",
//   intent="update",
//   rule="administrator",
//   resource_id="key",
//   description="Update blob metadata",
//   output="model.BlobMetaOutput",
// )
func (s *BlobsService) UpdateBlob(ctx context.Context, msg abstract.Message, input *model.BlobUpdateInput) (*abstract.Result, error) {
	return NewUpdateBlobHandler(s.blobStore)(ctx, msg)
}

// BeginUpload begins a resumable blob upload.
//
// @hestia.register(
//   name="system:blobs:blob:begin",
//   intent="create",
//   rule="administrator",
//   description="Begin a resumable blob upload",
//   output="model.UploadBeginOutput",
// )
func (s *BlobsService) BeginUpload(ctx context.Context, msg abstract.Message, input *model.BlobBeginInput) (*abstract.Result, error) {
	return NewBeginUploadHandler(s.blobStore, s.staging)(ctx, msg)
}

// UploadChunk uploads a chunk of a resumable blob upload.
//
// @hestia.register(
//   name="system:blobs:blob:chunk",
//   intent="create",
//   rule="administrator",
//   description="Upload a chunk of a resumable blob upload",
//   output="model.UploadChunkOutput",
// )
func (s *BlobsService) UploadChunk(ctx context.Context, msg abstract.Message, input *model.BlobChunkInput) (*abstract.Result, error) {
	return NewUploadChunkHandler(s.staging)(ctx, msg)
}

// CompleteUpload completes a resumable blob upload.
//
// @hestia.register(
//   name="system:blobs:blob:complete",
//   intent="create",
//   rule="administrator",
//   description="Complete a resumable blob upload",
//   output="model.BlobMetaOutput",
// )
func (s *BlobsService) CompleteUpload(ctx context.Context, msg abstract.Message, input *model.BlobCompleteInput) (*abstract.Result, error) {
	return NewCompleteUploadHandler(s.blobStore, s.staging)(ctx, msg)
}

// ProgressUpload reports progress of a resumable blob upload.
//
// @hestia.register(
//   name="system:blobs:blob:progress",
//   intent="query",
//   rule="administrator",
//   description="Report progress of a resumable blob upload",
//   output="model.UploadProgressOutput",
// )
func (s *BlobsService) ProgressUpload(ctx context.Context, msg abstract.Message, input *model.BlobProgressInput) (*abstract.Result, error) {
	return NewProgressUploadHandler(s.staging)(ctx, msg)
}

// AbortUpload aborts a resumable blob upload.
//
// @hestia.register(
//   name="system:blobs:blob:abort",
//   intent="create",
//   rule="administrator",
//   description="Abort a resumable blob upload",
// )
func (s *BlobsService) AbortUpload(ctx context.Context, msg abstract.Message, input *model.BlobAbortInput) (*abstract.Result, error) {
	return NewAbortUploadHandler(s.staging)(ctx, msg)
}

// RenameBlob renames a blob key.
//
// @hestia.register(
//
//	name="system:blobs:blob:rename",
//	intent="update",
//	rule="administrator",
//	resource_id="key",
//	description="Rename a blob key",
//
// )
func (s *BlobsService) RenameBlob(ctx context.Context, msg abstract.Message, input *model.BlobRenameInput) (*abstract.Result, error) {
	return NewRenameBlobHandler(s.blobStore)(ctx, msg)
}

// NamespaceStats gets stats for a blob namespace.
//
// @hestia.register(
//
//	name="system:blobs:namespace:stats",
//	intent="query",
//	rule="administrator",
//	description="Get namespace stats",
//	output="model.NamespaceStatsOutput",
//
// )
func (s *BlobsService) NamespaceStats(ctx context.Context, msg abstract.Message, input *model.NsInput) (*abstract.Result, error) {
	return NewStatsNamespaceHandler(s.blobStore)(ctx, msg)
}

// VerifyNamespace verifies blob integrity in a namespace.
//
// @hestia.register(
//
//	name="system:blobs:namespace:verify",
//	intent="query",
//	rule="administrator",
//	description="Verify namespace integrity",
//
// )
func (s *BlobsService) VerifyNamespace(ctx context.Context, msg abstract.Message, input *model.NsInput) (*abstract.Result, error) {
	return NewVerifyNamespaceHandler(s.blobStore)(ctx, msg)
}

// CompactNamespace compacts a namespace to reclaim space.
//
// @hestia.register(
//
//	name="system:blobs:namespace:compact",
//	intent="create",
//	rule="administrator",
//	description="Compact a namespace to reclaim space",
//	output="model.CompactResultOutput",
//
// )
func (s *BlobsService) CompactNamespace(ctx context.Context, msg abstract.Message, input *model.NsInput) (*abstract.Result, error) {
	return NewCompactNamespaceHandler(s.blobStore)(ctx, msg)
}


// @note #storage-leak-blobs-never-compact-ad27adbb issue resolved P1 #storage-leak,#blobs : Storage leak: blobs never compacted after deletion
// @assignee opencode
// Added Compact to BlobNamespace interface and nsHandle implementation. NewDeleteBlobHandler now triggers async compaction after successful deletion. Namespace deletion already closes the engine (automatic cleanup).
//
// When blobs or namespaces are deleted via system:blobs:blob:delete or system:blobs:namespace:delete,
// the physical segment files (.vol, .wal) remain on disk because Compact is never called.
// The blob library (github.com/asaidimu/blobs) relies on Compact to reclaim space from deleted chunks.
// Need to either: 1) call Compact after bulk deletions, 2) add a background compaction job, or 3) expose Compact via API for operators.
