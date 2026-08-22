package collections

import (
	"github.com/asaidimu/go-anansi/v8/core/document"
)

type CollectionMetaView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Name                   string         `anansi:"name"`
	Schema                 map[string]any `anansi:"schema"`
	Created                string         `anansi:"created"`
	Updated                string         `anansi:"updated"`
}

type CollectionDeletedView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Name                   string `anansi:"name"`
}

type CollectionPage struct {
	Collections []string `anansi:"collections"`
}

type CollectionListOutput struct {
	Page CollectionPage `anansi:"page"`
}

type CollectionOutput struct {
	Document CollectionMetaView `anansi:"document"`
}

type CollectionQueryOutput struct {
	Page map[string]any `anansi:"page"`
}

type CollectionDocumentOutput struct {
	Document CollectionDocumentView `anansi:"document"`
}

type CollectionDocumentView struct {
	ID   string         `anansi:"id"`
	Data map[string]any `anansi:"data"`
}
