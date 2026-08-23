package annotate

import (
	"go/ast"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseUsers(t *testing.T) {
	anns, err := Parse(filepath.Join("testdata", "users"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(anns) != 5 {
		t.Fatalf("got %d annotations, want 5", len(anns))
	}

	byName := map[string]Annotation{}
	for _, a := range anns {
		byName[a.MessageName] = a
	}

	t.Run("CreateUser", func(t *testing.T) {
		a, ok := byName["system:users:user:create"]
		if !ok {
			t.Fatal("missing create")
		}
		if a.Verb != VerbCreate || a.Rule != "administrator" {
			t.Errorf("verb/rule: %s / %s", a.Verb, a.Rule)
		}
		if a.MethodName != "CreateUser" || a.Service != "UsersService" {
			t.Errorf("method/service: %s / %s", a.MethodName, a.Service)
		}
		if !a.HasInput || a.HasStream {
			t.Errorf("hasInput=%v hasStream=%v", a.HasInput, a.HasStream)
		}
		if a.Input.Name != "CreateUserInput" || a.Input.AsWritten != "CreateUserInput" {
			t.Errorf("input type: %+v", a.Input)
		}
		if a.Result != ResultDocument {
			t.Errorf("result: %s, want document", a.Result)
		}
		if a.Output.Name != "User" {
			t.Errorf("output: %+v", a.Output)
		}
		if a.ResourceIDField != "id" {
			t.Errorf("resourceIDField: %q, want id", a.ResourceIDField)
		}
		if a.Description == "" {
			t.Error("description should be captured")
		}
	})

	t.Run("DeleteUser", func(t *testing.T) {
		a := byName["system:users:user:delete"]
		if a.Result != ResultEmpty {
			t.Errorf("result: %s, want empty", a.Result)
		}
		if a.ResourceIDField != "user_id" {
			t.Errorf("resourceIDField: %q, want user_id", a.ResourceIDField)
		}
		if a.Input.Name != "DeleteUserInput" {
			t.Errorf("input type: %+v", a.Input)
		}
	})

	t.Run("CheckUser no input", func(t *testing.T) {
		a := byName["system:users:user:check"]
		if a.HasInput {
			t.Error("CheckUser should have no input")
		}
		if a.Result != ResultEmpty {
			t.Errorf("result: %s", a.Result)
		}
		if a.ResourceIDField != "" {
			t.Errorf("resourceIDField: %q, want empty", a.ResourceIDField)
		}
	})

	t.Run("ListUsers documents result", func(t *testing.T) {
		a := byName["system:collections:user:query"]
		if a.Result != ResultDocuments {
			t.Errorf("result: %s, want documents", a.Result)
		}
		if a.Output.Name != "User" {
			t.Errorf("output: %+v", a.Output)
		}
		if a.ResourceIDField != "" {
			t.Errorf("resourceIDField: %q, want empty (no arguments tag)", a.ResourceIDField)
		}
	})

	t.Run("AckCallback fire and forget", func(t *testing.T) {
		a := byName["system:users:callback:ack"]
		if !a.FireAndForget {
			t.Error("FireAndForget should parse from fire_and_forget=\"true\"")
		}
		if a.Result != ResultEmpty {
			t.Errorf("result: %s, want empty", a.Result)
		}
		if a.Verb != VerbCreate || a.Rule != "authenticated" {
			t.Errorf("verb/rule: %s / %s", a.Verb, a.Rule)
		}
		// Absent attribute must default to false.
		for _, other := range anns {
			if other.MessageName == "system:users:callback:ack" {
				continue
			}
			if other.FireAndForget {
				t.Errorf("%s: FireAndForget must default to false", other.MessageName)
			}
		}
	})
}

func TestParseBadMissingName(t *testing.T) {
	_, err := Parse(filepath.Join("testdata", "bad"))
	if err == nil {
		t.Fatal("expected error for missing name attribute, got nil")
	}
	if len(err.Error()) == 0 {
		t.Fatal("empty error message")
	}
}

func TestParseStream(t *testing.T) {
	anns, err := Parse(filepath.Join("testdata", "feed"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(anns) != 1 {
		t.Fatalf("got %d annotations, want 1", len(anns))
	}
	a := anns[0]
	if !a.HasStream {
		t.Error("expected stream role")
	}
	if a.HasInput {
		t.Error("stream handler should not also have input role")
	}
	if a.Input.Name != "Event" || a.Input.AsWritten != "Event" {
		t.Errorf("stream item: %+v", a.Input)
	}
	if a.Result != ResultEmpty {
		t.Errorf("result: %s, want empty", a.Result)
	}
	if a.Verb != VerbStream {
		t.Errorf("verb: %s", a.Verb)
	}
}

func TestParseCrossPackageType(t *testing.T) {
	// dispatch.Item[Event] exercises the import-map resolution path for
	// selector types (the /dispatch suffix + IndexExpr item extraction).
	anns, err := Parse(filepath.Join("testdata", "feed"))
	if err != nil {
		t.Fatalf("Parse feed: %v", err)
	}
	if len(anns) == 0 {
		t.Fatal("no feed annotations")
	}
}

func TestParseRejectsDeprecatedHeaderFields(t *testing.T) {
	// header_fields was removed with abstract.Input.HeaderFields; transport
	// context bindings now come from input:"context.*" schema tags. Stale
	// declarations must fail codegen loudly instead of silently dropping the
	// binding.
	_, err := Parse(filepath.Join("testdata", "bad_header_fields"))
	if err == nil {
		t.Fatal("expected error for deprecated header_fields attribute, got nil")
	}
	if !strings.Contains(err.Error(), "header_fields") || !strings.Contains(err.Error(), "context") {
		t.Fatalf("error should name header_fields and the context.* migration, got: %v", err)
	}
}

func TestSplitPairsQuoteAware(t *testing.T) {
	got := splitPairs(`name="a,b",desc="c(d,e)",plain=z`)
	if len(got) != 3 {
		t.Fatalf("got %d parts, want 3: %q", len(got), got)
	}
	want := []string{`name="a,b"`, `desc="c(d,e)"`, "plain=z"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("part %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestAnnotationBlocksMultiple(t *testing.T) {
	// Regression: the depth-counter break in annotationBlocks exited the
	// switch, not the enclosing for, so end slid to the LAST ')' in the text
	// and every block after the first was merged into the first registration.
	// Each block must now produce its own entry.
	cg := &ast.CommentGroup{List: []*ast.Comment{
		{Text: "// @hestia.register(name = \"system:multi:one\", intent = CREATE)"},
		{Text: "// @hestia.register(name = \"system:multi:two\", intent = READ)"},
	}}
	blocks := annotationBlocks(cg)
	if len(blocks) != 2 {
		t.Fatalf("got %d blocks, want 2: %v", len(blocks), blocks)
	}
	want := []map[string]string{
		{"name": "system:multi:one", "intent": "CREATE"},
		{"name": "system:multi:two", "intent": "READ"},
	}
	for i, w := range want {
		for k, v := range w {
			if got := blocks[i][k]; got != v {
				t.Errorf("block %d %q = %q, want %q", i, k, got, v)
			}
		}
	}
}
