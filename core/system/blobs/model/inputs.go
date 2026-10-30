package model

type NsInput struct {
	NS string `input:"arguments.ns"`
}

type NsCreateInput struct {
	NS          string            `input:"arguments.ns"`
	DisplayName string            `input:"payload.display_name"`
	Public      bool              `input:"payload.public"`
	Custom      map[string]string `input:"payload.custom"`
}

type BlobKeyInput struct {
	NS  string `input:"arguments.ns"`
	Key string `input:"arguments.key"`
}

type BlobListInput struct {
	NS     string `input:"arguments.ns"`
	Prefix string `input:"payload.prefix"`
	Limit  int    `input:"payload.limit"`
}

type BlobUpdateInput struct {
	NS     string            `input:"arguments.ns"`
	Key    string            `input:"arguments.key"`
	Custom map[string]string `input:"payload.custom"`
}

type BlobUploadInput struct {
	NS          string `input:"arguments.ns"`
	Key         string `input:"arguments.key"`
	ContentType string `input:"context.content_type"`
	Overwrite   string `input:"modifiers.overwrite"`
	Payload     []byte `input:"payload"`
}

// @note #blob-20261029-008 todo P2 #blobs,#api,#ergonomics : No custom metadata can be attached at upload time
// @author claude
// @see #feat-ns-metadata
//
// store.PutCustom and the completed staging upload both carry a Custom map (complete passes cu.Custom), but nothing populates it: BlobBeginInput and BlobUploadInput have no custom field, begin only hands the staging manager a content type, size and block size, and the client's UploadOptions has no custom either. Callers must upload and then call update() as a second, non-atomic step, and a failure in between leaves a blob without its metadata.
// Proposal: accept payload.custom on begin and a custom form (query or header) on direct upload. The same gap exists for namespaces: #feat-ns-metadata.

type BlobBeginInput struct {
	NS          string `input:"arguments.ns"`
	Overwrite   string `input:"modifiers.overwrite"`
	Key         string `input:"payload.key"`
	Size        int64  `input:"payload.size"`
	ContentType string `input:"payload.content_type"`
	BlockSize   int64  `input:"payload.block_size"`
}

type BlobChunkInput struct {
	NS        string `input:"arguments.ns"`
	SessionID string `input:"context.session_id"`
	Offset    string `input:"context.offset"`
	SHA256    string `input:"context.chunk_sha256"`
	Payload   []byte `input:"payload"`
}

type BlobCompleteInput struct {
	NS        string `input:"arguments.ns"`
	SessionID string `input:"context.session_id"`
	Overwrite string `input:"modifiers.overwrite"`
}

type BlobAbortInput struct {
	NS        string `input:"arguments.ns"`
	SessionID string `input:"context.session_id"`
}

type BlobProgressInput struct {
	NS        string `input:"arguments.ns"`
	SessionID string `input:"modifiers.session_id"`
}

type BlobRenameInput struct {
	NS     string `input:"arguments.ns"`
	Key    string `input:"arguments.key"`
	NewKey string `input:"payload.new_key"`
}