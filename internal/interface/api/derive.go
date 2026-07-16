package api

import (
	"fmt"
	"strings"

	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	"github.com/asaidimu/hestia/app/core/registration"
)

func DeriveRoute(name string, arguments map[string]definition.FieldType) string {
	parts := strings.SplitN(name, ":", 4)
	path := fmt.Sprintf("/%s/%s/%s", parts[0], parts[1], parts[2])
	for arg := range arguments {
		path += fmt.Sprintf("/{%s}", arg)
	}
	return path
}

func IntentToHTTPMethod(verb registration.Verb) string {
	switch verb {
	case registration.Create:
		return "POST"
	case registration.Read:
		return "GET"
	case registration.Update:
		return "PATCH"
	case registration.Delete:
		return "DELETE"
	case registration.Query:
		return "POST"
	case registration.Stream:
		return "GET"
	}
	return "GET"
}

func IntentToHTTPPath(verb registration.Verb, path string) string {
	switch verb {
	case registration.Query:
		return path + "/query"
	case registration.Stream:
		return path + "/stream"
	}
	return path
}
