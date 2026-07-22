package api

import (
	"context"
	"encoding/json"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	"github.com/asaidimu/hestia/core/runtime"
)

// BuildInputDocument builds a data.Document from a request's path params,
// query string, and body, according to the input definition.
func BuildInputDocument(ctx context.Context, input runtime.Input, pathParams map[string]string, query map[string][]string, body []byte) *data.Document {
	doc := data.MustNewDocument(map[string]any{}, ctx)

	args := make(map[string]any)
	for _, argDef := range input.Arguments {
		if v, ok := pathParams[argDef.Name]; ok {
			args[argDef.Name] = v
		}
	}
	doc.Set("arguments", args)

	modifiers := make(map[string]any)
	for name := range input.Modifiers {
		if vals, ok := query[name]; ok && len(vals) > 0 {
			modifiers[name] = vals[0]
		}
	}
	doc.Set("modifiers", modifiers)

	if input.Payload != 0 {
		switch input.Payload {
		case definition.FieldTypeBytes:
			doc.Set("payload", body)
		default:
			var payload map[string]any
			if len(body) > 0 {
				if err := json.Unmarshal(body, &payload); err == nil {
					doc.Set("payload", payload)
				}
			}
		}
	}

	return doc
}

// SanitizeToMap returns a sanitized map representation of doc.
func SanitizeToMap(doc *data.Document) map[string]any {
	if doc == nil {
		return nil
	}
	sane, _ := doc.Sanitize()
	if sane == nil {
		return nil
	}
	return sane.ToMap()
}

// SanitizeAll returns sanitized map representations for all documents.
func SanitizeAll(docs data.DocumentSet) []any {
	items := make([]any, 0, len(docs))
	for _, d := range docs {
		if m := SanitizeToMap(d); m != nil {
			items = append(items, m)
		}
	}
	return items
}
