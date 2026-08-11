package collections

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

var (
	_collectionListOutput     = dispatch.MustFromJSON(collectionListOutputJSON)
	_collectionOutput         = dispatch.MustFromJSON(collectionOutputJSON)
	_collectionQueryOutput    = dispatch.MustFromJSON(collectionQueryOutputJSON)
	_collectionDocumentOutput = dispatch.MustFromJSON(collectionDocumentOutputJSON)
)

func collectionListOutputSchema() *definition.Schema     { return _collectionListOutput }
func collectionOutputSchema() *definition.Schema         { return _collectionOutput }
func collectionQueryOutputSchema() *definition.Schema    { return _collectionQueryOutput }
func collectionDocumentOutputSchema() *definition.Schema { return _collectionDocumentOutput }

var collectionListOutputJSON = []byte(`{
	"name": "collection_list_output",
	"description": "List of collections",
	"version": "1.0.0",
	"fields": {
		"page": {
			"name": "page",
			"description": "Paginated collection list",
			"type": "object",
			"schema": { "id": "collection_page" }
		}
	},
	"schemas": {
		"collection_page": {
			"name": "CollectionPage",
			"fields": {
				"collections": {
					"name": "collections",
					"description": "Array of collection names",
					"type": "array",
					"schema": { "type": "string" }
				}
			}
		}
	}
}`)

var collectionOutputJSON = []byte(`{
	"name": "collection_output",
	"description": "Collection schema definition",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"description": "Collection document",
			"type": "object",
			"schema": { "id": "collection_meta" }
		}
	},
	"schemas": {
		"collection_meta": {
			"name": "CollectionMeta",
			"fields": {
				"name": { "name": "name", "description": "Collection name", "type": "string" },
				"schema": { "name": "schema", "description": "Collection schema JSON", "type": "record" }
			}
		}
	}
}`)

var collectionQueryOutputJSON = []byte(`{
	"name": "collection_query_output",
	"description": "Paginated collection documents",
	"version": "1.0.0",
	"fields": {
		"page": { "name": "page", "description": "Paginated document results", "type": "record" }
	}
}`)

var collectionDocumentOutputJSON = []byte(`{
	"name": "collection_document_output",
	"description": "A document within a collection",
	"version": "1.0.0",
	"fields": {
		"document": {
			"name": "document",
			"description": "Collection document",
			"type": "object",
			"schema": { "id": "collection_document" }
		}
	},
	"schemas": {
		"collection_document": {
			"name": "CollectionDocument",
			"fields": {
				"id": { "name": "id", "description": "Document ID", "type": "string" },
				"data": { "name": "data", "description": "Document fields", "type": "record" }
			}
		}
	}
}`)

