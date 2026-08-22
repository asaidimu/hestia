package blobs

import (
	"github.com/asaidimu/go-anansi/v8/core/document"
)

type NamespaceView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	ID                     string `anansi:"id"`
	DisplayName            string `anansi:"display_name"`
	Public                 bool   `anansi:"public"`
}

type NamespaceListDocument struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Namespaces             []NamespaceView `anansi:"namespaces"`
}

type NamespaceListOutput struct {
	Document NamespaceListDocument `anansi:"document"`
}

type NamespaceOutput struct {
	Document NamespaceView `anansi:"document"`
}

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

type BlobListDocument struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Blobs                  []BlobMetaView `anansi:"blobs"`
}

type BlobListOutput struct {
	Document BlobListDocument `anansi:"document"`
}

type BlobMetaOutput struct {
	Document BlobMetaView `anansi:"document"`
}

type UploadBeginDocument struct {
	document.DocumentModel `json:"-" anansi:"-"`
	SessionID              string `anansi:"session_id"`
	Key                    string `anansi:"key"`
	Offset                 int64  `anansi:"offset"`
	BlockSize              int64  `anansi:"block_size"`
}

type UploadBeginOutput struct {
	Document UploadBeginDocument `anansi:"document"`
}

type UploadChunkDocument struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Total                  int64 `anansi:"total"`
}

type UploadChunkOutput struct {
	Document UploadChunkDocument `anansi:"document"`
}

type ByteRange struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Start                  int64 `anansi:"start"`
	End                    int64 `anansi:"end"`
}

type UploadProgressDocument struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Total                  int64       `anansi:"total"`
	Ranges                 []ByteRange `anansi:"ranges"`
	BlockSize              int64       `anansi:"block_size"`
	ExpectedSize           int64       `anansi:"expected_size"`
}

type UploadProgressOutput struct {
	Document UploadProgressDocument `anansi:"document"`
}
