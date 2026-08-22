package blobs

type NsInput struct {
	NS string `input:"arguments.ns"`
}

type NsCreateInput struct {
	NS          string `input:"arguments.ns"`
	DisplayName string `input:"payload.display_name"`
	Public      bool   `input:"payload.public"`
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
	ContentType string `input:"headers.content_type"`
	Overwrite   string `input:"modifiers.overwrite"`
	Payload     []byte `input:"payload"`
}

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
	SessionID string `input:"headers.session_id"`
	Offset    string `input:"headers.offset"`
	SHA256    string `input:"headers.sha256"`
	Payload   []byte `input:"payload"`
}

type BlobCompleteInput struct {
	NS        string `input:"arguments.ns"`
	SessionID string `input:"headers.session_id"`
	Overwrite string `input:"modifiers.overwrite"`
}

type BlobAbortInput struct {
	NS        string `input:"arguments.ns"`
	SessionID string `input:"headers.session_id"`
}

type BlobProgressInput struct {
	NS        string `input:"arguments.ns"`
	SessionID string `input:"modifiers.session_id"`
}
