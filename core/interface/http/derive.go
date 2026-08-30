package http

import (
	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime/route"
)

// DeriveRoute, IntentToHTTPMethod and IntentToHTTPPath moved to the
// transport-neutral core/runtime/route package (audit A-3). These wrappers
// keep the in-package call sites and tests stable; non-HTTP consumers
// import core/runtime/route directly.

func DeriveRoute(name string, arguments []abstract.ArgumentDefinition) string {
	return route.DeriveRoute(name, arguments)
}

func IntentToHTTPMethod(verb abstract.Verb) string {
	return route.IntentToHTTPMethod(verb)
}

func IntentToHTTPPath(verb abstract.Verb, path string) string {
	return route.IntentToHTTPPath(verb, path)
}
