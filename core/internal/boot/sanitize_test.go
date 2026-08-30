package boot

import (
	"testing"

	"go.uber.org/zap"

	"github.com/asaidimu/go-anansi/v8/core/sanitize"
)

// TestSanitizeConfig_GlobalIsPreserveOnly pins the global sanitization
// policy: it must carry no patterns and no field rules at all. Policies are
// type-blind (masked values are always strings), so any name-based global
// rule risks matching a non-string field of a user-defined collection and
// failing Document.Sanitize for whole documents — which silently empties
// query responses. See go-anansi devnote #sanitize-type-blindness.
func TestSanitizeConfig_GlobalIsPreserveOnly(t *testing.T) {
	cfg := sanitizeConfig()
	if cfg.Global == nil {
		t.Fatal("expected global sanitize config")
	}

	if len(cfg.Global.Patterns) != 0 {
		t.Errorf("global config must have no patterns, got %d", len(cfg.Global.Patterns))
	}
	if len(cfg.Global.Fields) != 0 {
		t.Errorf("global config must have no field rules, got %d", len(cfg.Global.Fields))
	}
	if cfg.Global.DefaultPolicy != sanitize.MaskPreserve {
		t.Errorf("global default policy must be MaskPreserve, got %q", cfg.Global.DefaultPolicy)
	}
}

// TestSanitizeConfig_NoScopedRules verifies that the global config no longer
// contains scoped rules. Scoped rules are now registered per-feature via
// allSanitizationRules in the system module.
func TestSanitizeConfig_NoScopedRules(t *testing.T) {
	cfg := sanitizeConfig()
	if len(cfg.Scoped) != 0 {
		t.Errorf("global config must have no scoped rules, got %d", len(cfg.Scoped))
	}
}

// TestSanitizeConfig_GlobalPreservesUnknownCollections exercises the real
// registry: with the preserve-only global policy, documents from unknown or
// user-defined collections — including array-typed fields that used to trip
// masking patterns (the `notes.authors` regression) — survive sanitization
// untouched.
func TestSanitizeConfig_GlobalPreservesUnknownCollections(t *testing.T) {
	sanitize.ResetForTesting()
	if err := sanitize.Configure(sanitizeConfig(), zap.NewNop()); err != nil {
		t.Fatalf("Configure: %v", err)
	}

	global := sanitize.GetScopedSanitizer("")
	if global == nil {
		t.Fatal("expected global sanitizer after Configure")
	}

	authors := []any{"alice", "bob"}
	out := global.SanitizeDocumentDeep(map[string]any{
		"password":     "hunter2",
		"access_token": "tok-abc",
		"api_key":      "key-xyz",
		"secret":       "s3cr3t",
		"hash":         "$2a$10$abc",
		"credential":   "crd",
		"authors":      authors,
		"name":         "visible",
	})
	for field, want := range map[string]any{
		"password":     "hunter2",
		"access_token": "tok-abc",
		"api_key":      "key-xyz",
		"secret":       "s3cr3t",
		"hash":         "$2a$10$abc",
		"credential":   "crd",
		"name":         "visible",
	} {
		if out[field] != want {
			t.Errorf("field %q should be preserved, got %v", field, out[field])
		}
	}
	if got, ok := out["authors"].([]any); !ok || len(got) != 2 || got[0] != "alice" {
		t.Errorf("authors array should survive intact, got %v", out["authors"])
	}
}

// TestSanitizeConfig_FeatureScopedRules tests that feature-scoped rules
// can be registered and used. This simulates what the system module does.
func TestSanitizeConfig_FeatureScopedRules(t *testing.T) {
	sanitize.ResetForTesting()
	if err := sanitize.Configure(sanitizeConfig(), zap.NewNop()); err != nil {
		t.Fatalf("Configure: %v", err)
	}

	// Register feature-scoped rules (simulating what system module does)
	reg := sanitize.Registry()
	_ = reg.Register("users", &sanitize.FieldMaskConfig{
		DefaultPolicy: sanitize.MaskPreserve,
		Fields: map[string]sanitize.MaskedFieldPolicy{
			"password": sanitize.MaskRedact,
		},
	})

	userScope := sanitize.GetScopedSanitizer("users")
	if userScope == nil {
		t.Fatal("expected users scoped sanitizer")
	}
	out := userScope.SanitizeDocumentDeep(map[string]any{
		"password": "hunter2",
		"email":    "a@b.com",
	})
	if out["password"] != "***" {
		t.Errorf("user password must be redacted, got %v", out["password"])
	}
	if out["email"] != "a@b.com" {
		t.Errorf("user email must be preserved, got %v", out["email"])
	}
}
