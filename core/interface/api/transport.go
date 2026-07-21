package api

import (
	"github.com/asaidimu/go-anansi/v8/core/common"

	"github.com/asaidimu/hestia/core/abstract"
)

type Request = abstract.Request
type Response = abstract.Response
type Cookie = abstract.Cookie
type StreamBody = abstract.StreamBody
type Handler = abstract.Handler
type Transport = abstract.Transport

var codeToStatus = map[string]int{
	"ERR_ACCESS_DENIED":     403,
	"NOT_FOUND":             404,
	"ALREADY_EXISTS":        409,
	"VALIDATION_ERROR":      400,
	"INVALID_REQUEST":       400,
	"UNAUTHORIZED":          401,
	"INVALID_CREDENTIALS":   401,
	"EMAIL_EXISTS":          409,
	"USER_DELETED":          410,
	"FORBIDDEN":             403,
	"MISSING_PARAM":         400,
	"INVALID_QDSL":          400,
	"DOCUMENT_REQUIRED":     400,
	"PARSE_DOCUMENT":        400,
	"SCHEMA_REQUIRED":       400,
	"SCHEMA_MISSING_NAME":   400,
	"COLLECTION_EXISTS":     409,
	"RESERVED_NAME":         409,
	"AUTH_REQUIRED":         401,
	"DOCUMENT_NOT_FOUND":    404,
	"NOT_IMPLEMENTED":       501,
	"SERVICE_UNAVAILABLE":   503,
}

func CodeToStatus(code string) int {
	if s, ok := codeToStatus[code]; ok {
		return s
	}
	return 500
}

func SystemErrorToStatus(err *common.SystemError) int {
	return CodeToStatus(err.Code)
}
