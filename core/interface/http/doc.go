package http

import (
	"bytes"
	"encoding/json"

	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	"github.com/asaidimu/hestia/core/runtime"
)

func rootField(s *definition.Schema, name string) bool {
	if s == nil {
		return false
	}
	_, f := s.FindField(name)
	return f != nil
}

// BuildInputDocument builds a schema-bound document from a request's path
// params, query string, and body by composing the input envelope as a single
// JSON byte slice — arguments and modifiers marshaled from tiny maps, the
// payload body embedded verbatim, header-sourced fields (e.g. blob upload
// content_type) injected — and decoding it once with the registration's pool.
// Decode errors (malformed body, type mismatch) surface as errors instead of
// being silently dropped.
func BuildInputDocument(pool *document.DocumentPool, input runtime.Input, req Request) (*document.Document, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')

	writeSection := func(name string, val []byte) {
		if buf.Len() > 1 {
			buf.WriteByte(',')
		}
		buf.WriteString(`"`)
		buf.WriteString(name)
		buf.WriteString(`":`)
		buf.Write(val)
	}

	if cts := req.Headers["Content-Type"]; len(cts) > 0 && rootField(input.Schema, "content_type") {
		ct, err := json.Marshal(cts[0])
		if err != nil {
			return nil, err
		}
		writeSection("content_type", ct)
	}

	if rootField(input.Schema, "arguments") {
		args := make(map[string]any)
		for _, argDef := range input.Arguments {
			if v, ok := req.PathParams[argDef.Name]; ok {
				args[argDef.Name] = v
			}
		}
		if len(args) > 0 {
			argsJSON, err := json.Marshal(args)
			if err != nil {
				return nil, err
			}
			writeSection("arguments", argsJSON)
		}
	}

	if rootField(input.Schema, "modifiers") {
		modifiers := make(map[string]any)
		for name := range input.Modifiers {
			if vals, ok := req.Query[name]; ok && len(vals) > 0 {
				modifiers[name] = vals[0]
			}
		}
		if len(modifiers) > 0 {
			modifiersJSON, err := json.Marshal(modifiers)
			if err != nil {
				return nil, err
			}
			writeSection("modifiers", modifiersJSON)
		}
	}

	if rootField(input.Schema, "payload") {
		if input.Payload == definition.FieldTypeBytes {
			p, err := json.Marshal(req.Body)
			if err != nil {
				return nil, err
			}
			writeSection("payload", p)
		} else if len(req.Body) > 0 {
			writeSection("payload", req.Body)
		}
	}

	buf.WriteByte('}')

	return pool.FromJSON(buf.Bytes())
}
