package model

import "github.com/asaidimu/go-anansi/v8/core/document"

// NamespaceView is the wire shape of a blob namespace.
type NamespaceView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	ID                     string `anansi:"id"`
	DisplayName            string `anansi:"display_name"`
	Public                 bool   `anansi:"public"`
}

// NamespaceListDocument is the body of a namespace list response.
type NamespaceListDocument struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Namespaces             []NamespaceView `anansi:"namespaces"`
}

// NamespaceListOutput is the envelope declaring the namespace list schema.
type NamespaceListOutput struct {
	Document NamespaceListDocument `anansi:"document"`
}

// NamespaceOutput is the envelope declaring the single-namespace schema.
type NamespaceOutput struct {
	Document NamespaceView `anansi:"document"`
}

// BlobMetaView is the wire shape of blob metadata. Timestamps are RFC3339
// strings to keep the derived schema's "string" field types stable.
type BlobMetaView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Key                    string            `anansi:"key"`
	NamespaceID            string            `anansi:"namespace_id"`
	ContentType            string            `anansi:"content_type"`
	Size                   int64             `anansi:"size"`
	CreatedAt              string            `anansi:"created_at"`
	UpdatedAt              *string           `anansi:"updated_at,omitempty"`
	Custom                 map[string]string `anansi:"custom"`
}

// BlobListDocument is the body of a blob list response.
type BlobListDocument struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Blobs                  []BlobMetaView `anansi:"blobs"`
}

// BlobListOutput is the envelope declaring the blob list schema.
type BlobListOutput struct {
	Document BlobListDocument `anansi:"document"`
}

// BlobMetaOutput is the envelope declaring the blob metadata schema.
type BlobMetaOutput struct {
	Document BlobMetaView `anansi:"document"`
}

// UploadBeginDocument is the body of a begin-upload response.
type UploadBeginDocument struct {
	document.DocumentModel `json:"-" anansi:"-"`
	SessionID              string `anansi:"session_id"`
	Key                    string `anansi:"key"`
	Offset                 int64  `anansi:"offset"`
	BlockSize              int64  `anansi:"block_size"`
}

// UploadBeginOutput is the envelope declaring the begin-upload schema.
type UploadBeginOutput struct {
	Document UploadBeginDocument `anansi:"document"`
}

// UploadChunkDocument is the body of a chunk-write response.
type UploadChunkDocument struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Total                  int64 `anansi:"total"`
}

// UploadChunkOutput is the envelope declaring the chunk-write schema.
type UploadChunkOutput struct {
	Document UploadChunkDocument `anansi:"document"`
}

// ByteRange is a half-open [Start, End) byte range.
type ByteRange struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Start                  int64 `anansi:"start"`
	End                    int64 `anansi:"end"`
}

// UploadProgressDocument is the body of a progress response.
type UploadProgressDocument struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Total                  int64       `anansi:"total"`
	Ranges                 []ByteRange `anansi:"ranges"`
	BlockSize              int64       `anansi:"block_size"`
	ExpectedSize           int64       `anansi:"expected_size"`
}

// UploadProgressOutput is the envelope declaring the progress schema.
type UploadProgressOutput struct {
	Document UploadProgressDocument `anansi:"document"`
}

// NamespaceStatsDocument is the body of a namespace stats response.
type NamespaceStatsDocument struct {
	document.DocumentModel `json:"-" anansi:"-"`
	NamespaceID            string `anansi:"namespace_id"`
	BlobCount              int64  `anansi:"blob_count"`
	BytesStored            int64  `anansi:"bytes_stored"`
	BytesPhysical          int64  `anansi:"bytes_physical"`
	ChunkCount             int64  `anansi:"chunk_count"`
	DeadBytes              int64  `anansi:"dead_bytes"`
	DeadChunks             int64  `anansi:"dead_chunks"`
	SegmentCount           int64  `anansi:"segment_count"`
}

// NamespaceStatsOutput is the envelope declaring the namespace stats schema.
type NamespaceStatsOutput struct {
	Document NamespaceStatsDocument `anansi:"document"`
}

// CompactResultDocument is the body of a compact response.
type CompactResultDocument struct {
	document.DocumentModel `json:"-" anansi:"-"`
	BlobsRemoved           int64 `anansi:"blobs_removed"`
	ChunksRemoved          int64 `anansi:"chunks_removed"`
	BytesFreed             int64 `anansi:"bytes_freed"`
	SegmentsCompacted      int64 `anansi:"segments_compacted"`
}

// CompactResultOutput is the envelope declaring the compact result schema.
type CompactResultOutput struct {
	Document CompactResultDocument `anansi:"document"`
}