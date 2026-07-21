package collections

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/hestia/core/schema"
)

var (
	_collectionListOutput       = schema.MustFromJSON(collectionListOutputJSON)
	_collectionOutput           = schema.MustFromJSON(collectionOutputJSON)
	_collectionQueryOutput      = schema.MustFromJSON(collectionQueryOutputJSON)
	_collectionDocumentOutput   = schema.MustFromJSON(collectionDocumentOutputJSON)
	_collectionGetInput         = schema.MustFromJSON(collectionGetInputJSON)
	_collectionCreateInput      = schema.MustFromJSON(collectionCreateInputJSON)
	_collectionDeleteInput      = schema.MustFromJSON(collectionDeleteInputJSON)
	_collectionDocQueryInput    = schema.MustFromJSON(collectionDocQueryInputJSON)
	_collectionDocCreateInput   = schema.MustFromJSON(collectionDocCreateInputJSON)
	_collectionDocGetInput      = schema.MustFromJSON(collectionDocGetInputJSON)
	_collectionDocUpdateInput   = schema.MustFromJSON(collectionDocUpdateInputJSON)
	_collectionDocDeleteInput   = schema.MustFromJSON(collectionDocDeleteInputJSON)
)

func collectionListOutputSchema() *definition.Schema     { return _collectionListOutput }
func collectionOutputSchema() *definition.Schema         { return _collectionOutput }
func collectionQueryOutputSchema() *definition.Schema    { return _collectionQueryOutput }
func collectionDocumentOutputSchema() *definition.Schema { return _collectionDocumentOutput }
func collectionGetInputSchema() *definition.Schema       { return _collectionGetInput }
func collectionCreateInputSchema() *definition.Schema    { return _collectionCreateInput }
func collectionDeleteInputSchema() *definition.Schema    { return _collectionDeleteInput }
func collectionDocQueryInputSchema() *definition.Schema  { return _collectionDocQueryInput }
func collectionDocCreateInputSchema() *definition.Schema { return _collectionDocCreateInput }
func collectionDocGetInputSchema() *definition.Schema    { return _collectionDocGetInput }
func collectionDocUpdateInputSchema() *definition.Schema { return _collectionDocUpdateInput }
func collectionDocDeleteInputSchema() *definition.Schema { return _collectionDocDeleteInput }

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

var collectionGetInputJSON = []byte(`{
	"name": "collection_get_input",
	"description": "Collection name from path",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"description": "Collection name argument",
			"type": "object",
			"schema": { "id": "collection_get_args" }
		}
	},
	"schemas": {
		"collection_get_args": {
			"name": "CollectionGetArgs",
			"fields": {
				"name": { "name": "name", "description": "Collection name", "type": "string" }
			}
		}
	}
}`)

var collectionCreateInputJSON = []byte(`{
	"name": "collection_create_input",
	"description": "Anansi schema definition for the new collection",
	"version": "1.0.0",
	"fields": {
		"payload": {
			"name": "payload",
			"description": "Anansi schema JSON definition",
			"type": "record"
		}
	}
}`)

var collectionDeleteInputJSON = []byte(`{
	"name": "collection_delete_input",
	"description": "Collection name from path",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"description": "Collection name argument",
			"type": "object",
			"schema": { "id": "collection_delete_args" }
		}
	},
	"schemas": {
		"collection_delete_args": {
			"name": "CollectionDeleteArgs",
			"fields": {
				"name": { "name": "name", "description": "Collection name", "type": "string" }
			}
		}
	}
}`)

var collectionDocQueryInputJSON = []byte(`{
	"name": "collection_doc_query_input",
	"description": "QDSL query with collection name from path",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"description": "Collection name argument",
			"type": "object",
			"schema": { "id": "collection_doc_query_args" }
		},
		"payload": {
			"name": "payload",
			"description": "QDSL query object",
			"type": "record"
		}
	},
	"schemas": {
		"collection_doc_query_args": {
			"name": "CollectionDocQueryArgs",
			"fields": {
				"name": { "name": "name", "description": "Collection name", "type": "string" }
			}
		}
	}
}`)

var collectionDocCreateInputJSON = []byte(`{
	"name": "collection_doc_create_input",
	"description": "Create a document in a collection",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"description": "Collection name argument",
			"type": "object",
			"schema": { "id": "collection_doc_create_args" }
		},
		"payload": {
			"name": "payload",
			"description": "Document data",
			"type": "record"
		}
	},
	"schemas": {
		"collection_doc_create_args": {
			"name": "CollectionDocCreateArgs",
			"fields": {
				"name": { "name": "name", "description": "Collection name", "type": "string" }
			}
		}
	}
}`)

var collectionDocGetInputJSON = []byte(`{
	"name": "collection_doc_get_input",
	"description": "Get a document by collection and document ID",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"description": "Collection name and document ID arguments",
			"type": "object",
			"schema": { "id": "collection_doc_get_args" }
		}
	},
	"schemas": {
		"collection_doc_get_args": {
			"name": "CollectionDocGetArgs",
			"fields": {
				"name": { "name": "name", "description": "Collection name", "type": "string" },
				"doc_id": { "name": "doc_id", "description": "Document ID", "type": "string" }
			}
		}
	}
}`)

var collectionDocUpdateInputJSON = []byte(`{
	"name": "collection_doc_update_input",
	"description": "Update a document by collection and document ID",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"description": "Collection name and document ID arguments",
			"type": "object",
			"schema": { "id": "collection_doc_update_args" }
		},
		"payload": {
			"name": "payload",
			"description": "Updated document fields",
			"type": "record"
		}
	},
	"schemas": {
		"collection_doc_update_args": {
			"name": "CollectionDocUpdateArgs",
			"fields": {
				"name": { "name": "name", "description": "Collection name", "type": "string" },
				"doc_id": { "name": "doc_id", "description": "Document ID", "type": "string" }
			}
		}
	}
}`)

var collectionDocDeleteInputJSON = []byte(`{
	"name": "collection_doc_delete_input",
	"description": "Delete a document by collection and document ID",
	"version": "1.0.0",
	"fields": {
		"arguments": {
			"name": "arguments",
			"description": "Collection name and document ID arguments",
			"type": "object",
			"schema": { "id": "collection_doc_delete_args" }
		}
	},
	"schemas": {
		"collection_doc_delete_args": {
			"name": "CollectionDocDeleteArgs",
			"fields": {
				"name": { "name": "name", "description": "Collection name", "type": "string" },
				"doc_id": { "name": "doc_id", "description": "Document ID", "type": "string" }
			}
		}
	}
}`)
