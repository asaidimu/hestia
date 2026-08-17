package boot

import (
	"testing"

	"go.uber.org/zap"

	"github.com/asaidimu/go-anansi/v8/core/sanitize"
)

// TestSanitizeConfig_SecretPatternsRedact pins the global sanitization policy:
// every secret-bearing field name (password, hash, secret, token, api_key,
// credential) is redacted by pattern, while auth-shaped values are hashed.
func TestSanitizeConfig_SecretPatternsRedact(t *testing.T) {
	cfg := sanitizeConfig()
	if cfg.Global == nil {
		t.Fatal("expected global sanitize config")
	}

	patternPolicy := map[string]sanitize.MaskedFieldPolicy{}
	for _, p := range cfg.Global.Patterns {
		patternPolicy[p.Pattern] = p.Policy
	}

	expectRedact := []string{
		`(?i)password`,
		`(?i)hash`,
		`(?i)secret`,
		`(?i)token`,
		`(?i)api[_-]?key`,
		`(?i)credential`,
	}
	for _, expr := range expectRedact {
		if patternPolicy[expr] != sanitize.MaskRedact {
			t.Errorf("pattern %q must be MaskRedact, got %q", expr, patternPolicy[expr])
		}
	}
	if patternPolicy[`(?i)auth`] != sanitize.MaskHash {
		t.Errorf("pattern (?i)auth must be MaskHash, got %q", patternPolicy[`(?i)auth`])
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
}

// TestSanitizeConfig_AppliesToDocuments exercises the real registry: after
// Configure, sensitive fields are masked while safe fields survive.
func TestSanitizeConfig_AppliesToDocuments(t *testing.T) {
	sanitize.ResetForTesting()
	if err := sanitize.Configure(sanitizeConfig(), zap.NewNop()); err != nil {
		t.Fatalf("Configure: %v", err)
	}

	global := sanitize.GetScopedSanitizer("")
	if global == nil {
		t.Fatal("expected global sanitizer after Configure")
	}

	out := global.SanitizeDocumentDeep(map[string]any{
		"password":     "hunter2",
		"access_token": "tok-abc",
		"api_key":      "key-xyz",
		"secret":       "s3cr3t",
		"hash":         "$2a$10$abc",
		"credential":   "crd",
		"token_version": int64(3),
		"name":         "visible",
	})
	for _, field := range []string{"password", "access_token", "api_key", "secret", "hash", "credential"} {
		if out[field] != "***" {
			t.Errorf("field %q should be redacted, got %v", field, out[field])
		}
	}
	// token_version is explicitly preserved (it is not secret).
	if out["token_version"] != int64(3) {
		t.Errorf("token_version should be preserved, got %v", out["token_version"])
	}
	if out["name"] != "visible" {
		t.Errorf("name should be preserved, got %v", out["name"])
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