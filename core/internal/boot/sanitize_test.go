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

// TestSanitizeConfig_ScopedOverrides pins the per-collection exceptions: user
// passwords are redacted but emails preserved; api key hashes redacted.
func TestSanitizeConfig_ScopedOverrides(t *testing.T) {
	cfg := sanitizeConfig()

	userCfg := cfg.Scoped["_user_"]
	if userCfg == nil {
		t.Fatal("expected _user_ scoped config")
	}
	if userCfg.Fields["password"] != sanitize.MaskRedact {
		t.Errorf("_user_.password must be MaskRedact, got %q", userCfg.Fields["password"])
	}
	if userCfg.Fields["email"] != sanitize.MaskPreserve {
		t.Errorf("_user_.email must be MaskPreserve, got %q", userCfg.Fields["email"])
	}
	if userCfg.Fields["token_version"] != sanitize.MaskPreserve {
		t.Errorf("_user_.token_version must be MaskPreserve, got %q", userCfg.Fields["token_version"])
	}

	keyCfg := cfg.Scoped["_api_key_"]
	if keyCfg == nil {
		t.Fatal("expected _api_key_ scoped config")
	}
	if keyCfg.Fields["hash"] != sanitize.MaskRedact {
		t.Errorf("_api_key_.hash must be MaskRedact, got %q", keyCfg.Fields["hash"])
	}

	logCfg := cfg.Scoped["_access_log_"]
	if logCfg == nil {
		t.Fatal("expected _access_log_ scoped config")
	}
	if logCfg.Fields["integrity_hash"] != sanitize.MaskRedact {
		t.Errorf("_access_log_.integrity_hash must be MaskRedact, got %q", logCfg.Fields["integrity_hash"])
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

// TestSanitizeConfig_UserScopeMasksPasswordPreservesEmail exercises the
// scoped override through the registry.
func TestSanitizeConfig_UserScopeMasksPasswordPreservesEmail(t *testing.T) {
	sanitize.ResetForTesting()
	if err := sanitize.Configure(sanitizeConfig(), zap.NewNop()); err != nil {
		t.Fatalf("Configure: %v", err)
	}

	userScope := sanitize.GetScopedSanitizer("_user_")
	if userScope == nil {
		t.Fatal("expected _user_ scoped sanitizer")
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
