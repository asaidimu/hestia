package runtime

import "github.com/asaidimu/go-anansi/v8/core/common"

// Common SystemError sentinels.
// Use chainable fluent API to add context:
//
//	return core.ErrAccessDenied.
//		WithOperation("users:user:query").
//		WithCause(internalErr)
var (
	ErrAccessDenied         = common.NewSystemError("ERR_ACCESS_DENIED", "access denied")
	ErrNotFound             = common.NewSystemError("NOT_FOUND")
	ErrAlreadyExists        = common.NewSystemError("ALREADY_EXISTS")
	ErrValidation           = common.NewSystemError("VALIDATION_ERROR")
	ErrInvalidRequest       = common.NewSystemError("INVALID_REQUEST")
	ErrUnauthorized         = common.NewSystemError("UNAUTHORIZED")
	ErrInvalidCredentials   = common.NewSystemError("INVALID_CREDENTIALS")
	ErrInternal             = common.NewSystemError("INTERNAL_ERROR")
	ErrNotImplemented       = common.NewSystemError("NOT_IMPLEMENTED")
	ErrServiceUnavailable   = common.NewSystemError("SERVICE_UNAVAILABLE")
	ErrSchemaRequired       = common.NewSystemError("SCHEMA_REQUIRED")
	ErrSchemaMissingName    = common.NewSystemError("SCHEMA_MISSING_NAME")
	ErrCollectionExists     = common.NewSystemError("COLLECTION_EXISTS")
	ErrReservedName         = common.NewSystemError("RESERVED_NAME")
	ErrDocumentRequired     = common.NewSystemError("DOCUMENT_REQUIRED")
	ErrParseDocument        = common.NewSystemError("PARSE_DOCUMENT")
	ErrDocumentNotFound     = common.NewSystemError("DOCUMENT_NOT_FOUND")
	ErrAuthRequired         = common.NewSystemError("AUTH_REQUIRED")
	ErrMissingParam         = common.NewSystemError("MISSING_PARAM")
	ErrInvalidQDSL          = common.NewSystemError("INVALID_QDSL")
	ErrEmailExists          = common.NewSystemError("EMAIL_EXISTS")
	ErrUserDeleted          = common.NewSystemError("USER_DELETED")
	ErrForbidden            = common.NewSystemError("FORBIDDEN")
	ErrPermissionNotRegistered = common.NewSystemError("PERMISSION_NOT_REGISTERED")
	ErrOperationLacksPolicy   = common.NewSystemError("OPERATION_LACKS_POLICY")
	ErrInvalidToken           = common.NewSystemError("ERR_INVALID_TOKEN")
)
