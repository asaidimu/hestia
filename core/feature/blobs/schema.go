package blobs

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

var (
	_nsListOutput         = dispatch.MustFromJSON(nsListOutputJSON)
	_nsOutput             = dispatch.MustFromJSON(nsOutputJSON)
	_blobListOutput       = dispatch.MustFromJSON(blobListOutputJSON)
	_blobMetaOutput       = dispatch.MustFromJSON(blobMetaOutputJSON)
	_uploadBeginOutput    = dispatch.MustFromJSON(uploadBeginOutputJSON)
	_uploadChunkOutput    = dispatch.MustFromJSON(uploadChunkOutputJSON)
	_uploadProgressOutput = dispatch.MustFromJSON(uploadProgressOutputJSON)
)

func nsListOutputSchema() *definition.Schema   { return _nsListOutput }
func nsOutputSchema() *definition.Schema       { return _nsOutput }
func blobListOutputSchema() *definition.Schema { return _blobListOutput }
func blobMetaOutputSchema() *definition.Schema { return _blobMetaOutput }
func uploadBeginOutputSchema() *definition.Schema {
	return _uploadBeginOutput
}
func uploadChunkOutputSchema() *definition.Schema { return _uploadChunkOutput }
func uploadProgressOutputSchema() *definition.Schema {
	return _uploadProgressOutput
}

var nsListOutputJSON = []byte(`{
	"name": "blob_namespace_list",
	"description": "List of blob namespaces",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"description": "Namespace list document",
			"type": "object",
			"schema": { "id": "ns_list_document" }
		}
	},
	"schemas": {
		"ns_list_document": {
			"name": "NamespaceListDocument",
			"fields": {
				"namespaces": {
					"name": "namespaces",
					"description": "Array of blob namespaces",
					"type": "array",
					"schema": { "id": "namespace" }
				}
			}
		},
		"namespace": {
			"name": "Namespace",
			"fields": {
				"id": { "name": "id", "description": "Namespace ID", "type": "string" },
				"display_name": { "name": "display_name", "description": "Display name", "type": "string" },
				"public": { "name": "public", "description": "Whether public access is enabled", "type": "boolean" }
			}
		}
	}
}`)

var nsOutputJSON = []byte(`{
	"name": "blob_namespace",
	"description": "A blob namespace",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"description": "Namespace document",
			"type": "object",
			"schema": { "id": "namespace" }
		}
	},
	"schemas": {
		"namespace": {
			"name": "Namespace",
			"fields": {
				"id": { "name": "id", "description": "Namespace ID", "type": "string" },
				"display_name": { "name": "display_name", "description": "Display name", "type": "string" },
				"public": { "name": "public", "description": "Whether public access is enabled", "type": "boolean" }
			}
		}
	}
}`)

var blobListOutputJSON = []byte(`{
	"name": "blob_list",
	"description": "List of blobs in a namespace",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"description": "Blob list document",
			"type": "object",
			"schema": { "id": "blob_list_document" }
		}
	},
	"schemas": {
		"blob_list_document": {
			"name": "BlobListDocument",
			"fields": {
				"blobs": {
					"name": "blobs",
					"description": "Array of blob metadata",
					"type": "array",
					"schema": { "id": "blob_meta" }
				}
			}
		},
		"blob_meta": {
			"name": "BlobMeta",
			"fields": {
				"key": { "name": "key", "description": "Blob key", "type": "string" },
				"namespace_id": { "name": "namespace_id", "description": "Namespace ID", "type": "string" },
				"content_type": { "name": "content_type", "description": "MIME content type", "type": "string" },
				"size": { "name": "size", "description": "Size in bytes", "type": "integer" },
				"created_at": { "name": "created_at", "description": "Creation timestamp", "type": "string" },
				"updated_at": { "name": "updated_at", "description": "Last modification timestamp", "type": "string" },
				"custom": { "name": "custom", "description": "Arbitrary metadata", "type": "object" }
			}
		}
	}
}`)

var blobMetaOutputJSON = []byte(`{
	"name": "blob_meta",
	"description": "Blob metadata",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"description": "Blob metadata document",
			"type": "object",
			"schema": { "id": "blob_meta" }
		}
	},
	"schemas": {
		"blob_meta": {
			"name": "BlobMeta",
			"fields": {
				"key": { "name": "key", "description": "Blob key", "type": "string" },
				"namespace_id": { "name": "namespace_id", "description": "Namespace ID", "type": "string" },
				"content_type": { "name": "content_type", "description": "MIME content type", "type": "string" },
				"size": { "name": "size", "description": "Size in bytes", "type": "integer" },
				"created_at": { "name": "created_at", "description": "Creation timestamp", "type": "string" },
				"updated_at": { "name": "updated_at", "description": "Last modification timestamp", "type": "string" },
				"custom": { "name": "custom", "description": "Arbitrary metadata", "type": "object" }
			}
		}
	}
}`)

var uploadBeginOutputJSON = []byte(`{
	"name": "upload_begin",
	"description": "Resumable upload session",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"description": "Upload session document",
			"type": "object",
			"schema": { "id": "upload_begin_document" }
		}
	},
	"schemas": {
		"upload_begin_document": {
			"name": "UploadBeginDocument",
			"fields": {
				"session_id": { "name": "session_id", "description": "Upload session ID", "type": "string" },
				"key": { "name": "key", "description": "Blob key", "type": "string" },
				"offset": { "name": "offset", "description": "Initial write offset", "type": "integer" },
				"block_size": { "name": "block_size", "description": "Upload block size", "type": "integer" }
			}
		}
	}
}`)

var uploadChunkOutputJSON = []byte(`{
	"name": "upload_chunk",
	"description": "Chunk write result",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"description": "Chunk write result document",
			"type": "object",
			"schema": { "id": "upload_chunk_document" }
		}
	},
	"schemas": {
		"upload_chunk_document": {
			"name": "UploadChunkDocument",
			"fields": {
				"total": { "name": "total", "description": "Total bytes received", "type": "integer" }
			}
		}
	}
}`)

var uploadProgressOutputJSON = []byte(`{
	"name": "upload_progress",
	"description": "Resumable upload progress",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"description": "Upload progress document",
			"type": "object",
			"schema": { "id": "upload_progress_document" }
		}
	},
	"schemas": {
		"upload_progress_document": {
			"name": "UploadProgressDocument",
			"fields": {
				"total": { "name": "total", "description": "Total bytes received", "type": "integer" },
				"ranges": {
					"name": "ranges",
					"description": "Received byte ranges",
					"type": "array",
					"schema": { "id": "byte_range" }
				},
				"block_size": { "name": "block_size", "description": "Upload block size", "type": "integer" },
				"expected_size": { "name": "expected_size", "description": "Expected total size", "type": "integer" }
			}
		},
		"byte_range": {
			"name": "ByteRange",
			"fields": {
				"start": { "name": "start", "description": "Range start (inclusive)", "type": "integer" },
				"end": { "name": "end", "description": "Range end (exclusive)", "type": "integer" }
			}
		}
	}
}`)

