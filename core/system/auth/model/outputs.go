package model

import (
	"github.com/asaidimu/go-anansi/v8/core/document"
)

// LoginDocumentView is the wire shape of a login response document. The token
// itself is carried in the session cookie, so only the user is emitted here.
type LoginDocumentView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Token                  string         `anansi:"token"`
	User                   map[string]any `anansi:"user,omitempty"`
}

// LoginOutput declares the login response schema.
type LoginOutput struct {
	Document LoginDocumentView `anansi:"document"`
}

// MessageOutput declares a simple status message response.
type MessageOutput struct {
	Message string `anansi:"message"`
}

// ElevateDocumentView is the wire shape of a privilege elevation response.
type ElevateDocumentView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	Key                    string `anansi:"key"`
}

// ElevateOutput declares the elevation response schema.
type ElevateOutput struct {
	Document ElevateDocumentView `anansi:"document"`
}

// ClaimsDocumentView is the wire shape of a validated session token's claims.
type ClaimsDocumentView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	UserID                 string `anansi:"user_id"`
	SessionID              string `anansi:"session_id"`
	IssuedAt               int64  `anansi:"issued_at"`
	ExpiresAt              int64  `anansi:"expires_at"`
	CreatedAt              int64  `anansi:"created_at"`
}

// APIKeyClaimsView is the wire shape of a validated API key's claims.
type APIKeyClaimsView struct {
	document.DocumentModel `json:"-" anansi:"-"`
	UserID                 string   `anansi:"user_id"`
	Email                  string   `anansi:"email"`
	Permissions            []string `anansi:"permissions"`
	TokenType              string   `anansi:"token_type"`
	TokenID                string   `anansi:"token_id,omitempty"`
	ExpiresAt              int64    `anansi:"expires_at,omitempty"`
}

// ClaimsOutput declares the claims response schema.
type ClaimsOutput struct {
	Document ClaimsDocumentView `anansi:"document"`
}