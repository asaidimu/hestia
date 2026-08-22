package model

import (
	"github.com/asaidimu/go-anansi/v8/core/document"
)

type APIKeyListInput struct{}

type APIKeyGetInput struct {
	KeyID string `input:"arguments.key_id"`
}

type APIKeyDeleteInput struct {
	KeyID string `input:"arguments.key_id"`
}

type APIKeyRotateInput struct {
	KeyID string `input:"arguments.key_id"`
}

type APIKeyOutput struct {
	Document APIKeyPublic `anansi:"document"`
}

type APIKeyCreatedOutput struct {
	document.DocumentModel `json:"-" anansi:"-"`
	APIKeyPublic
	Key string `anansi:"key,required=false" json:"key,omitempty"`
}

type APIKeyListOutput struct {
	Documents []APIKeyPublic `anansi:"documents"`
}
