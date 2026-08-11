package blobs

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

var (
	_nsListOutput   = dispatch.MustFromJSON(nsListOutputJSON)
	_nsOutput       = dispatch.MustFromJSON(nsOutputJSON)
	_blobListOutput = dispatch.MustFromJSON(blobListOutputJSON)
	_blobMetaOutput = dispatch.MustFromJSON(blobMetaOutputJSON)
)

func nsListOutputSchema() *definition.Schema   { return _nsListOutput }
func nsOutputSchema() *definition.Schema       { return _nsOutput }
func blobListOutputSchema() *definition.Schema { return _blobListOutput }
func blobMetaOutputSchema() *definition.Schema { return _blobMetaOutput }

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

