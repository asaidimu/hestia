package model

import (
	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

// APIKeyListInput binds the dispatch input document's arguments for listing keys.
type APIKeyListInput struct {
	UserID string `input:"arguments.user_id"`
}

// APIKeyGetInput binds the dispatch input document's arguments into a key ID.
type APIKeyGetInput struct {
	KeyID string `input:"arguments.key_id"`
}

// APIKeyDeleteInput binds the dispatch input document's arguments into a key ID.
type APIKeyDeleteInput struct {
	KeyID string `input:"arguments.key_id"`
}

// APIKeyRotateInput binds the dispatch input document's arguments into a key ID.
type APIKeyRotateInput struct {
	KeyID string `input:"arguments.key_id"`
}

// APIKeyOutput is the single-key response contract, exposing the hash-free
// APIKeyPublic projection.
type APIKeyOutput struct {
	Document APIKeyPublic `anansi:"document"`
}

// APIKeyCreatedOutput is the create/rotate response contract: the public
// projection plus the raw key material, which is only revealed once.
type APIKeyCreatedOutput struct {
	document.DocumentModel `json:"-" anansi:"-"`
	APIKeyPublic
	Key string `anansi:"key,required=false" json:"key,omitempty"`
}

// APIKeyListOutput is the key list response contract.
type APIKeyListOutput struct {
	Documents []APIKeyPublic `anansi:"documents"`
}

func APIKeyListInputSchema() *definition.Schema {
	return dispatch.SchemaFromTypeWithTag[APIKeyListInput]("input")
}
func APIKeyGetInputSchema() *definition.Schema {
	return dispatch.SchemaFromTypeWithTag[APIKeyGetInput]("input")
}
func APIKeyDeleteInputSchema() *definition.Schema {
	return dispatch.SchemaFromTypeWithTag[APIKeyDeleteInput]("input")
}
func APIKeyRotateInputSchema() *definition.Schema {
	return dispatch.SchemaFromTypeWithTag[APIKeyRotateInput]("input")
}
func APIKeyCreateInputSchema() *definition.Schema {
	return dispatch.SchemaFromTypeWithTag[APIKeyCreate]("input", true)
}
func APIKeyUpdateInputSchema() *definition.Schema {
	return dispatch.SchemaFromTypeWithTag[APIKeyUpdate]("input", true)
}
func APIKeyOutputSchema() *definition.Schema {
	return dispatch.SchemaFromType[APIKeyOutput]()
}
func APIKeyListOutputSchema() *definition.Schema {
	return dispatch.SchemaFromType[APIKeyListOutput]()
}
