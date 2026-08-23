// Package annotate parses @hestia.register(...) doc-comment blocks on service
// methods and introspects the free-form handler signature into a structured
// Annotation. It is the front end of the registrations/policies generator
// (Step 3): the parser answers "what is registered, with what contract", the
// generator answers "what code to emit".
//
// Signature grammar (each recognized parameter role is resolved by type):
//
//	context.Context          → ctx
//	abstract.Message         → msg
//	*TIn                     → input (bound via "input" tag)
//	<-chan dispatch.Item[TIn] → stream
//
// Results:
//
//	(error)                 → empty result (HandleEmpty)
//	(TOut, error)           → document result (HandleDocument / HandleDocuments)
//
// Type references are resolved against the file's import map so the generator
// can emit qualified references (e.g. model.SystemUser) without re-guessing.
package annotate

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// Verb is the message intent as declared in the annotation block. It mirrors
// the subset of abstract.Verb used by registrations without forcing the parser
// to import the runtime package (keeps parsing pure / fast).
type Verb string

const (
	VerbCreate Verb = "create"
	VerbRead   Verb = "read"
	VerbUpdate Verb = "update"
	VerbDelete Verb = "delete"
	VerbQuery  Verb = "query"
	VerbStream Verb = "stream"
	VerbCheck  Verb = "check"
)

// ResultKind classifies the method's return shape.
type ResultKind string

const (
	ResultEmpty     ResultKind = "empty"
	ResultDocument  ResultKind = "document"
	ResultDocuments ResultKind = "documents"
	// ResultRaw marks a method that returns (*abstract.Result, error) directly,
	// bypassing the dispatch document-wrapping adapters. Used by handlers that
	// must set Result fields the adapters don't expose (e.g. SessionToken).
	ResultRaw ResultKind = "raw"
)

// TypeRef is a type reference resolved against the file's import map.
type TypeRef struct {
	// AsWritten is the source form, e.g. "users.CreateUserInput" or
	// "CreateUserInput" for a same-package type.
	AsWritten string
	// Name is the short identifier, e.g. "CreateUserInput".
	Name string
	// ImportPath is the full package import path, empty for same-package types.
	ImportPath string
	// Package is the import alias used in source (the importMap key), empty
	// for same-package types. Kept as-written so aliased imports (e.g.
	// usersmodel ".../users/model") render qualified references that compile.
	Package string
	// Star is true when the type appeared as *T.
	Star bool
}

// Annotation is the structured form of one @hestia.register block.
type Annotation struct {
	// Declared attributes from the doc-comment block.
	MessageName string // e.g. "system:users:user:create"
	Verb        Verb
	Rule        string // policy rule key, e.g. "administrator"
	Description string
	// ResourceIDField is the resource identity for route registration,
	// declared via resource_id="...". Empty when absent.
	ResourceIDField string
	// HeaderFields maps an HTTP header to the input schema field it lands in,
	// declared via header_fields="Header=field,...". Empty when absent.
	HeaderFields map[string]string
	// BootstrapSafe marks the registration boot-safe (no DB required), declared
	// via bootstrap_safe="true". False when absent.
	BootstrapSafe bool
	// Internal marks the registration internal (not routed over HTTP), declared
	// via internal="true". False when absent.
	Internal bool
	// FireAndForget marks the registration fire-and-forget (transports accept
	// the message and respond immediately, e.g. HTTP 202 with the message ID),
	// declared via fire_and_forget="true". False when absent.
	FireAndForget bool
	// OutputAttr is the output type declared via output="Type", used when the
	// signature alone can't express the Output schema (raw and empty-result
	// handlers that still declare a wire output). Empty when absent.
	OutputAttr TypeRef

	// Method info.
	MethodName string // e.g. "CreateUser"
	Service    string // receiver type name, e.g. "UsersService"

	// Signature-derived contract.
	HasInput bool // *TIn present
	HasStream bool // <-chan dispatch.Item[TIn] present
	Input    TypeRef  // TIn for input/stream
	Result   ResultKind
	Output   TypeRef // TOut when Result is document/documents
}

// Parse parses all .go files in dir (skipping _test.go) and returns every
// @hestia.register annotation found on service methods.
func Parse(dir string) ([]Annotation, error) {
	fset := token.NewFileSet()
	var files []*ast.File
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}

	var out []Annotation
	for _, f := range files {
		imports := importMap(f)
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Doc == nil || fd.Recv == nil {
				continue
			}
			service, ok := receiverService(fd)
			if !ok {
				continue
			}
			for _, block := range annotationBlocks(fd.Doc) {
				ann, err := buildAnnotation(fset, fd, service, block, imports)
				if err != nil {
					return nil, fmt.Errorf("%s: %v", fset.Position(fd.Pos()), err)
				}
				out = append(out, ann)
			}
		}
	}
	return out, nil
}

// importMap builds alias → import path from a file's import declarations.
func importMap(f *ast.File) map[string]string {
	m := map[string]string{}
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		name := filepath.Base(path)
		if imp.Name != nil && imp.Name.Name != "_" {
			name = imp.Name.Name
		}
		m[name] = path
	}
	return m
}

// receiverService returns the receiver type name if it is a service type
// (ends with "Service").
func receiverService(fd *ast.FuncDecl) (string, bool) {
	if len(fd.Recv.List) == 0 {
		return "", false
	}
	t := fd.Recv.List[0].Type
	switch tt := t.(type) {
	case *ast.StarExpr:
		t = tt.X
	}
	id, ok := t.(*ast.Ident)
	if !ok {
		return "", false
	}
	if !strings.HasSuffix(id.Name, "Service") {
		return "", false
	}
	return id.Name, true
}

// annotationBlocks extracts every @hestia.register(...) block from a doc
// comment group. Blocks may span multiple // lines, so the group is read as a
// single text blob and balanced parens are tracked across newlines. Each block
// is a list of key="value" pairs.
func annotationBlocks(doc *ast.CommentGroup) []map[string]string {
	text := doc.Text()
	var blocks []map[string]string
	idx := 0
	for {
		start := strings.Index(text[idx:], "@hestia.register")
		if start < 0 {
			break
		}
		start += idx
		open := strings.Index(text[start:], "(")
		if open < 0 {
			break
		}
		open += start

		depth := 0
		end := -1
	scan:
		for i := open; i < len(text); i++ {
			switch text[i] {
			case '(':
				depth++
			case ')':
				depth--
				if depth == 0 {
					end = i
					break scan
				}
			}
		}
		if end < 0 {
			break
		}
		inner := text[open+1 : end]
		blocks = append(blocks, parsePairs(inner))
		idx = end + 1
	}
	return blocks
}

// parsePairs parses key="value", key=value pairs separated by commas.
func parsePairs(s string) map[string]string {
	pairs := map[string]string{}
	for _, part := range splitPairs(s) {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		val = strings.Trim(val, `"`)
		pairs[key] = val
	}
	return pairs
}

func splitPairs(s string) []string {
	var parts []string
	var depth int
	var inString bool
	var cur strings.Builder
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch ch {
		case '"':
			inString = !inString
		case '(':
			if !inString {
				depth++
			}
		case ')':
			if !inString {
				depth--
			}
		case ',':
			if depth == 0 && !inString {
				parts = append(parts, strings.TrimSpace(cur.String()))
				cur.Reset()
				continue
			}
		}
		cur.WriteByte(ch)
	}
	if rest := strings.TrimSpace(cur.String()); rest != "" {
		parts = append(parts, rest)
	}
	return parts
}

func buildAnnotation(fset *token.FileSet, fd *ast.FuncDecl, service string, attrs map[string]string, imports map[string]string) (Annotation, error) {
	ann := Annotation{
		MessageName:     attrs["name"],
		Verb:            Verb(attrs["intent"]),
		Rule:            attrs["rule"],
		Description:     attrs["description"],
		ResourceIDField: attrs["resource_id"],
		HeaderFields:    parseHeaderFields(attrs["header_fields"]),
		MethodName:      fd.Name.Name,
		Service:         service,
	}
	ann.BootstrapSafe = parseBool(attrs["bootstrap_safe"])
	ann.Internal = parseBool(attrs["internal"])
	ann.FireAndForget = parseBool(attrs["fire_and_forget"])
	if ann.MessageName == "" {
		return ann, fmt.Errorf("@hestia.register missing required attribute name")
	}
	switch ann.Verb {
	case VerbCreate, VerbRead, VerbUpdate, VerbDelete, VerbQuery, VerbStream, VerbCheck:
	default:
		return ann, fmt.Errorf("@hestia.register missing or invalid attribute intent (got %q)", attrs["intent"])
	}

	if fd.Type.Params != nil {
		for _, f := range fd.Type.Params.List {
			role, ref, stream, err := classifyParam(f, imports)
			if err != nil {
				return ann, err
			}
			switch role {
			case "input":
				ann.HasInput = true
				ann.Input = ref
			case "stream":
				ann.HasStream = true
				ann.Input = ref
			}
			_ = stream
		}
	}

	ann.Result = ResultEmpty
	if fd.Type.Results != nil && len(fd.Type.Results.List) > 0 {
		outs := fd.Type.Results.List
		last := outs[len(outs)-1]
		if !isErrorType(last.Type) {
			return ann, fmt.Errorf("last result of %s must be error", fd.Name.Name)
		}
		if len(outs) == 1 {
			ann.Result = ResultEmpty
		} else if len(outs) == 2 {
			ref, isSlice, err := outputType(outs[0].Type, imports)
			if err != nil {
				return ann, err
			}
			ann.Output = ref
			if isSlice {
				ann.Result = ResultDocuments
			} else if isRawResult(ref) {
				ann.Result = ResultRaw
			} else {
				ann.Result = ResultDocument
			}
		} else {
			return ann, fmt.Errorf("%s: too many results (%d)", fd.Name.Name, len(outs))
		}
	}

	if declared := attrs["output"]; declared != "" {
		ref, err := attrTypeRef(declared, imports)
		if err != nil {
			return ann, fmt.Errorf("%s: output attribute: %v", fd.Name.Name, err)
		}
		ann.OutputAttr = ref
	}

	return ann, nil
}

// isRawResult reports whether a type reference is *abstract.Result, which the
// dispatch adapters can't wrap (it carries extra Result fields like
// SessionToken) and must be returned as-is.
func isRawResult(ref TypeRef) bool {
	return ref.Name == "Result" && ref.ImportPath != "" && strings.HasSuffix(ref.ImportPath, "/abstract")
}

// attrTypeRef resolves an output="Type" declaration against the import map,
// accepting both same-package names ("MessageOutput") and imported package
// names ("model.MessageOutput").
func attrTypeRef(declared string, imports map[string]string) (TypeRef, error) {
	parts := strings.Split(declared, ".")
	if len(parts) > 2 {
		return TypeRef{}, fmt.Errorf("unsupported type reference %q", declared)
	}
	if len(parts) == 1 {
		return TypeRef{AsWritten: declared, Name: declared}, nil
	}
	alias, name := parts[0], parts[1]
	path := imports[alias]
	if path == "" {
		return TypeRef{}, fmt.Errorf("cannot resolve package %q for %q", alias, declared)
	}
	return TypeRef{
		AsWritten:  declared,
		Name:       name,
		ImportPath: path,
		Package:    alias,
	}, nil
}

// classifyParam maps a parameter type to a role. ctx and msg are recognized
// and ignored; *TIn → input; <-chan dispatch.Item[TIn] → stream.
func classifyParam(f *ast.Field, imports map[string]string) (role string, ref TypeRef, stream bool, err error) {
	t := f.Type
	switch tt := t.(type) {
	case *ast.StarExpr:
		ref, err = typeRef(tt.X, imports)
		if err != nil {
			return "", ref, false, err
		}
		return "input", ref, false, nil
	case *ast.ChanType:
		if tt.Dir == ast.RECV {
			if ref, ok := streamItem(tt.Value, imports); ok {
				return "stream", ref, true, nil
			}
		}
		return "", TypeRef{}, false, fmt.Errorf("unsupported channel parameter %s", exprString(t))
	case *ast.SelectorExpr:
		sel := tt.Sel.Name
		alias := identName(tt.X)
		path := imports[alias]
		if path == "context" && sel == "Context" {
			return "ctx", TypeRef{}, false, nil
		}
		if strings.HasSuffix(path, "/abstract") && sel == "Message" {
			return "msg", TypeRef{}, false, nil
		}
		return "", TypeRef{}, false, fmt.Errorf("unsupported parameter %s (%s.%s)", exprString(t), path, sel)
	default:
		return "", TypeRef{}, false, fmt.Errorf("unsupported parameter type %s", exprString(t))
	}
}

// streamItem recognizes <-chan dispatch.Item[T] and returns the item type T.
func streamItem(v ast.Expr, imports map[string]string) (TypeRef, bool) {
	idx, ok := v.(*ast.IndexExpr)
	if !ok {
		return TypeRef{}, false
	}
	sel, ok := idx.X.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Item" {
		return TypeRef{}, false
	}
	alias := identName(sel.X)
	path := imports[alias]
	if !strings.HasSuffix(path, "/dispatch") {
		return TypeRef{}, false
	}
	ref, err := typeRef(idx.Index, imports)
	if err != nil {
		return TypeRef{}, false
	}
	ref.AsWritten = exprString(idx.Index)
	return ref, true
}

func outputType(t ast.Expr, imports map[string]string) (TypeRef, bool, error) {
	if arr, ok := t.(*ast.ArrayType); ok {
		ref, err := typeRef(arr.Elt, imports)
		return ref, true, err
	}
	ref, err := typeRef(t, imports)
	return ref, false, err
}

func isErrorType(t ast.Expr) bool {
	id, ok := t.(*ast.Ident)
	return ok && id.Name == "error"
}

// typeRef resolves an expression to a TypeRef against the import map.
func typeRef(t ast.Expr, imports map[string]string) (TypeRef, error) {
	asWritten := exprString(t)
	switch tt := t.(type) {
	case *ast.Ident:
		return TypeRef{AsWritten: asWritten, Name: tt.Name}, nil
	case *ast.StarExpr:
		ref, err := typeRef(tt.X, imports)
		ref.Star = true
		return ref, err
	case *ast.SelectorExpr:
		alias := identName(tt.X)
		path := imports[alias]
		if path == "" {
			return TypeRef{}, fmt.Errorf("cannot resolve package %q for %s", alias, asWritten)
		}
		return TypeRef{
			AsWritten:  asWritten,
			Name:       tt.Sel.Name,
			ImportPath: path,
			Package:    alias,
		}, nil
	default:
		return TypeRef{}, fmt.Errorf("unsupported type reference %s", asWritten)
	}
}

// parseHeaderFields parses a comma-separated Header=field list into the
// HeaderFields map. Empty/blank input yields nil.
func parseHeaderFields(s string) map[string]string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	fields := map[string]string{}
	for _, part := range strings.Split(s, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		header := strings.TrimSpace(kv[0])
		field := strings.Trim(strings.TrimSpace(kv[1]), `"`)
		if header != "" {
			fields[header] = field
		}
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}

// parseBool interprets a truthy attribute value as true.
func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func identName(e ast.Expr) string {
	id, ok := e.(*ast.Ident)
	if !ok {
		return ""
	}
	return id.Name
}

func exprString(t ast.Expr) string {
	switch tt := t.(type) {
	case *ast.Ident:
		return tt.Name
	case *ast.SelectorExpr:
		return identName(tt.X) + "." + tt.Sel.Name
	case *ast.StarExpr:
		return "*" + exprString(tt.X)
	case *ast.ChanType:
		dir := ""
		switch tt.Dir {
		case ast.SEND:
			dir = "chan<- "
		case ast.RECV:
			dir = "<-chan "
		default:
			dir = "chan "
		}
		return dir + exprString(tt.Value)
	case *ast.ArrayType:
		return "[]" + exprString(tt.Elt)
	default:
		return fmt.Sprintf("%T", t)
	}
}
