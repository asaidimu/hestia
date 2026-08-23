package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

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

// requestHeaderValue returns the first value of the named request header,
// matching case-insensitively. fasthttp canonicalizes header names it
// receives (X-Session-ID → X-Session-Id), while registrations declare them
// in their canonical, human-readable form, so an exact map lookup would
// silently drop headers like X-Session-ID and X-Chunk-SHA256.
func requestHeaderValue(headers map[string][]string, name string) (string, bool) {
	if vals, ok := headers[name]; ok && len(vals) > 0 {
		return vals[0], true
	}
	canonical := http.CanonicalHeaderKey(name)
	if vals, ok := headers[canonical]; ok && len(vals) > 0 {
		return vals[0], true
	}
	for k, vals := range headers {
		if len(vals) > 0 && strings.EqualFold(k, name) {
			return vals[0], true
		}
	}
	return "", false
}

// @note #perf-20260821-004 issue status=open priority=P0 tags=#performance,#correctness : BuildInputDocument uses unnecessary JSON round-trip
//
// BuildInputDocument manually constructs a JSON string by marshaling each
// field, concatenating into a JSON object, then parsing it back into a
// document via pool.FromJSON. This is:
//
// 1. Wasteful: marshal Go values → JSON bytes → concatenate → parse JSON
// 2. Incorrect for bytes: json.Marshal([]byte) base64-encodes, double-encoding
// 3. Error-prone: manual JSON construction can produce invalid JSON
//
// The anansi Document has direct Set* methods:
//
//	doc, _ := pool.New()
//	doc.SetString("arguments.name", value)
//	doc.SetBytes("payload", req.Body)
//
// This should populate fields directly without JSON:
//  1. pool.New() to get an empty document
//  2. For context: iterate input.ContextFields(), set each from req.Headers
//     via contextHeaderCandidates
//  3. For arguments: iterate input.Arguments(), set each from req.PathParams
//  4. For modifiers: iterate input.Modifiers(), set each from req.Query
//  5. For payload: if bytes, doc.SetBytes(); if JSON, use DecodeJSONField
//
// Note: Do NOT duplicate JSON decode logic. The anansi decoder (go-anansi/
// core/encoding/json) is already optimized. Use DecodeJSONField for JSON
// payloads — it decodes a single field directly into the document without
// building an intermediate JSON object.
//
// Impact:
// - Eliminates marshal/parse round-trip (2x CPU savings)
// - Eliminates base64 encoding for byte payloads (33% memory savings)
// - Eliminates intermediate JSON buffer allocation
// - For IoT/HFT: Critical for reducing per-request latency
//
// Resolution: Rewrite BuildInputDocument to use pool.New() and doc.Set*()
// methods directly, using DecodeJSONField only for JSON payload types.
// contextHeaderCandidates returns the deterministic candidate list for
// lifting a declared context field from request headers, most specific
// standard form first: "content_type" → ["Content-Type", "X-Content-Type"].
// The standard spelling wins when a client supplies both; the X-prefixed
// form covers custom transport headers like X-Session-ID. Matching is done
// case-insensitively by requestHeaderValue.
func contextHeaderCandidates(field string) []string {
	kebab := toKebabTitle(field)
	return []string{kebab, "X-" + kebab}
}

// toKebabTitle converts a snake_case schema field name to canonical HTTP
// header casing: "session_id" → "Session-Id", "sha256" → "Sha256",
// "chunk_sha256" → "Chunk-Sha256".
func toKebabTitle(name string) string {
	parts := strings.Split(name, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "-")
}

// BuildInputDocument assembles the dispatch document from the request:
//
//   - context.*  : lifted from request headers via contextHeaderCandidates
//   - arguments  : from path parameters
//   - modifiers  : from query parameters
//   - payload    : raw body (JSON or bytes per schema)
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

	if fields := input.ContextFields(); len(fields) > 0 && rootField(input.Schema, "context") {
		contextVals := make(map[string]any)
		for _, field := range fields {
			for _, candidate := range contextHeaderCandidates(field) {
				if v, ok := requestHeaderValue(req.Headers, candidate); ok {
					contextVals[field] = v
					break
				}
			}
		}
		if len(contextVals) > 0 {
			ctxJSON, err := json.Marshal(contextVals)
			if err != nil {
				return nil, err
			}
			writeSection("context", ctxJSON)
		}
	}

	if rootField(input.Schema, "arguments") {
		args := make(map[string]any)
		for _, argDef := range input.Arguments() {
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
		for name := range input.Modifiers() {
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
		if input.Payload() == definition.FieldTypeBytes {
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
