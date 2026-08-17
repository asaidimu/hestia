package collections

import (
	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

// CollectionMetaView is the wire shape of a collection's metadata.
type CollectionMetaView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Name                   string         `anansi:"name"`
	Schema                 map[string]any `anansi:"schema"`
	Created                string         `anansi:"created"`
	Updated                string         `anansi:"updated"`
}

// CollectionDeletedView is the wire shape of a delete-collection response.
type CollectionDeletedView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Name                   string `anansi:"name"`
}

// CollectionPage is the declared inner shape of the collection list page.
type CollectionPage struct {
	Collections []string `anansi:"collections"`
}

// CollectionListOutput is the envelope declaring the collection list schema.
type CollectionListOutput struct {
	Page CollectionPage `anansi:"page"`
}

func collectionListOutputSchema() *definition.Schema { return dispatch.SchemaFromType[CollectionListOutput]() }

// CollectionOutput is the envelope declaring the single-collection schema.
type CollectionOutput struct {
	Document CollectionMetaView `anansi:"document"`
}

func collectionOutputSchema() *definition.Schema { return dispatch.SchemaFromType[CollectionOutput]() }

// CollectionQueryOutput declares the page envelope of a collection query.
type CollectionQueryOutput struct {
	Page map[string]any `anansi:"page"`
}

func collectionQueryOutputSchema() *definition.Schema { return dispatch.SchemaFromType[CollectionQueryOutput]() }

// CollectionDocumentOutput declares the envelope of a single collection
// document response. The body is the raw persisted document.
type CollectionDocumentOutput struct {
	Document CollectionDocumentView `anansi:"document"`
}

// CollectionDocumentView is the declared shape of a collection document.
type CollectionDocumentView struct {
	ID   string         `anansi:"id"`
	Data map[string]any `anansi:"data"`
}

func collectionDocumentOutputSchema() *definition.Schema { return dispatch.SchemaFromType[CollectionDocumentOutput]() }
