package http

import (
	"fmt"
	"strings"

	"github.com/asaidimu/hestia/core/abstract"
)

func DeriveRoute(name string, arguments []abstract.ArgumentDefinition) string {
	parts := strings.Split(name, ":")
	path := "/" + strings.Join(parts, "/")
	for _, arg := range arguments {
		path += fmt.Sprintf("/{%s}", arg.Name)
	}
	return path
}

func IntentToHTTPMethod(verb abstract.Verb) string {
	switch verb {
	case abstract.Create:
		return "POST"
	case abstract.Read:
		return "GET"
	case abstract.Update:
		return "PATCH"
	case abstract.Delete:
		return "DELETE"
	case abstract.Query:
		return "POST"
	case abstract.Stream:
		return "GET"
	case abstract.Check:
		return "POST"
	}
	return "GET"
}

func IntentToHTTPPath(verb abstract.Verb, path string) string {
	return path
}
