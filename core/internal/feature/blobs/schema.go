package blobs

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/hestia/core/schema"
)

var (
	_nsInput          = schema.MustFromJSON(nsInputJSON)
	_nsCreateInput    = schema.MustFromJSON(nsCreateInputJSON)
	_blobKeyInput     = schema.MustFromJSON(blobKeyInputJSON)
	_blobListInput    = schema.MustFromJSON(blobListInputJSON)
	_nsListOutput     = schema.MustFromJSON(nsListOutputJSON)
	_nsOutput         = schema.MustFromJSON(nsOutputJSON)
	_blobListOutput   = schema.MustFromJSON(blobListOutputJSON)
	_blobMetaOutput   = schema.MustFromJSON(blobMetaOutputJSON)
	_blobUpdateInput  = schema.MustFromJSON(blobUpdateInputJSON)
)

func nsInputSchema() *definition.Schema           { return _nsInput }
func nsCreateInputSchema() *definition.Schema     { return _nsCreateInput }
func blobKeyInputSchema() *definition.Schema      { return _blobKeyInput }
func blobListInputSchema() *definition.Schema      { return _blobListInput }
func blobUpdateInputSchema() *definition.Schema    { return _blobUpdateInput }
func nsListOutputSchema() *definition.Schema       { return _nsListOutput }
func nsOutputSchema() *definition.Schema           { return _nsOutput }
func blobListOutputSchema() *definition.Schema     { return _blobListOutput }
func blobMetaOutputSchema() *definition.Schema     { return _blobMetaOutput }

var nsInputJSON = []byte(`{
	"name": "blob_ns_input",
	"description": "Namespace identifier from path",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"type": "object",
			"schema": { "id": "blob_ns_input_args" }
		}
	},
	"schemas": {
		"blob_ns_input_args": {
			"name": "BlobNsInputArgs",
			"fields": {
				"ns": { "name": "ns", "description": "Namespace ID", "type": "string" }
			}
		}
	}
}`)

var nsCreateInputJSON = []byte(`{
	"name": "blob_ns_create_input",
	"description": "Namespace creation request",
	"version": "1.0.0",
	"fields": {
		"payload": {
			"name": "payload",
			"type": "object",
			"schema": { "id": "blob_ns_create_payload" }
		}
	},
	"schemas": {
		"blob_ns_create_payload": {
			"name": "BlobNSCreatePayload",
			"fields": {
				"display_name": { "name": "display_name", "description": "Display name for the namespace", "type": "string" },
				"public": { "name": "public", "description": "Whether the namespace allows public (unauthenticated) access", "type": "boolean" }
			}
		}
	}
}`)

var blobKeyInputJSON = []byte(`{
	"name": "blob_key_input",
	"description": "Blob key and namespace from path",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"type": "object",
			"schema": { "id": "blob_key_input_args" }
		}
	},
	"schemas": {
		"blob_key_input_args": {
			"name": "BlobKeyInputArgs",
			"fields": {
				"ns": { "name": "ns", "description": "Namespace ID", "type": "string" },
				"key": { "name": "key", "description": "Blob key", "type": "string" }
			}
		}
	}
}`)

var blobListInputJSON = []byte(`{
	"name": "blob_list_input",
	"description": "List blobs with optional filters",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"type": "object",
			"schema": { "id": "blob_list_input_args" }
		},
		"payload": {
			"name": "payload",
			"type": "record",
			"schema": { "id": "blob_list_payload" }
		}
	},
	"schemas": {
		"blob_list_input_args": {
			"name": "BlobListInputArgs",
			"fields": {
				"ns": { "name": "ns", "description": "Namespace ID", "type": "string" }
			}
		},
		"blob_list_payload": {
			"name": "BlobListPayload",
			"fields": {
				"prefix": { "name": "prefix", "description": "Key prefix filter", "type": "string" },
				"limit": { "name": "limit", "description": "Max results", "type": "integer" }
			}
		}
	}
}`)

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

var blobUpdateInputJSON = []byte(`{
	"name": "blob_update_input",
	"description": "Update blob metadata",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"type": "object",
			"schema": { "id": "blob_update_input_args" }
		},
		"payload": {
			"name": "payload",
			"type": "object",
			"schema": { "id": "blob_update_payload" }
		}
	},
	"schemas": {
		"blob_update_input_args": {
			"name": "BlobUpdateInputArgs",
			"fields": {
				"ns": { "name": "ns", "description": "Namespace ID", "type": "string" },
				"key": { "name": "key", "description": "Blob key", "type": "string" }
			}
		},
		"blob_update_payload": {
			"name": "BlobUpdatePayload",
			"fields": {
				"content_type": { "name": "content_type", "description": "New MIME content type", "type": "string" },
				"custom": { "name": "custom", "description": "Arbitrary metadata to set", "type": "object" }
			}
		}
	}
}`)
