package http

import (
	"fmt"
	"strings"

	"github.com/asaidimu/hestia/core/abstract"
)

func DeriveRoute(name string, arguments []abstract.ArgDef) string {
	parts := strings.SplitN(name, ":", 4)
	path := fmt.Sprintf("/%s/%s/%s", parts[0], parts[1], parts[2])
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
	switch verb {
	case abstract.Query:
		return path + "/query"
	case abstract.Stream:
		return path + "/stream"
	case abstract.Check:
		return path + "/check"
	}
	return path
}
