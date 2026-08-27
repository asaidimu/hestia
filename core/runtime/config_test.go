package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

// TestApplyEnvOverridesWinsOverSetupConfig verifies todo/general_todo.md: env
// vars override hardcoded SetupConfig values.
// NOTE: Version is intentionally NOT in this list — it is compile-time only
// (-ldflags) and must not be overridable via environment so that a freshly
// swapped binary always reports its own baked-in version, not the old one.
func TestApplyEnvOverridesWinsOverSetupConfig(t *testing.T) {
	t.Setenv("PORT", "4321")
	t.Setenv("SESSION_SECRET", "env-secret")
	t.Setenv("DB_PATH", "/tmp/env.db")
	t.Setenv("API_PREFIX", "/v1")

	conf := DefaultConfig()
	conf.Version = "1.0.0"
	conf.Port = 9999
	conf.SessionSecret = "code-secret"
	conf.DBPath = "/code/db"
	conf.APIPrefix = "/api"

	if err := ApplyEnvOverrides(conf); err != nil {
		t.Fatalf("apply overrides: %v", err)
	}
	// Version must NOT be overridden by env — compile-time only.
	if conf.Version != "1.0.0" {
		t.Errorf("Version = %q, want hardcoded 1.0.0 (env must not override version)", conf.Version)
	}
	if conf.Port != 4321 {
		t.Errorf("Port = %d, want env 4321", conf.Port)
	}
	if conf.SessionSecret != "env-secret" {
		t.Errorf("SessionSecret = %q, want env-secret", conf.SessionSecret)
	}
	if conf.DBPath != "/tmp/env.db" {
		t.Errorf("DBPath = %q, want /tmp/env.db", conf.DBPath)
	}
	if conf.APIPrefix != "/v1" {
		t.Errorf("APIPrefix = %q, want /v1", conf.APIPrefix)
	}
}

// TestApplyEnvOverridesLeavesAbsentKnobs verifies that missing env vars do not
// clobber hardcoded values.
func TestApplyEnvOverridesLeavesAbsentKnobs(t *testing.T) {
	// Simulate absence of every overridable knob (the ambient test env may set
	// some of these).
	t.Setenv("PORT", "")
	t.Setenv("SESSION_SECRET", "")
	t.Setenv("DB_PATH", "")
	t.Setenv("API_PREFIX", "")
	t.Setenv("UPDATE_ENABLED", "")
	t.Setenv("UPDATE_GITHUB_OWNER", "")
	t.Setenv("UPDATE_SERVER_URL", "")

	conf := DefaultConfig()
	conf.Version = "1.0.0"
	conf.Port = 9999
	conf.SessionSecret = "code-secret"
	conf.DBPath = "/code/db"
	conf.APIPrefix = "/api"

	if err := ApplyEnvOverrides(conf); err != nil {
		t.Fatalf("apply overrides: %v", err)
	}
	if conf.Port != 9999 || conf.Version != "1.0.0" || conf.SessionSecret != "code-secret" ||
		conf.DBPath != "/code/db" || conf.APIPrefix != "/api" {
		t.Fatalf("hardcoded values clobbered: %+v", conf)
	}
	if conf.SelfUpdate != nil {
		t.Fatalf("SelfUpdate should stay nil without UPDATE_* env, got %+v", conf.SelfUpdate)
	}
}

func TestApplySelfUpdateEnvGithubProvider(t *testing.T) {
	t.Setenv("UPDATE_ENABLED", "true")
	t.Setenv("UPDATE_GITHUB_OWNER", "acme")
	t.Setenv("UPDATE_GITHUB_REPO", "app")
	t.Setenv("UPDATE_GITHUB_ASSET_PATTERN", "app-{{ .version }}-{{ .os }}_{{ .arch }}")
	t.Setenv("UPDATE_CHECK_SCHEDULE", "@every 12h")
	t.Setenv("UPDATE_AUTO_APPLY", "true")
	t.Setenv("UPDATE_SYSTEMD", "true")

	conf := DefaultConfig()
	if err := ApplyEnvOverrides(conf); err != nil {
		t.Fatalf("apply overrides: %v", err)
	}
	if conf.SelfUpdate == nil {
		t.Fatal("expected SelfUpdate to be configured from env")
	}
	if conf.SelfUpdate.Provider == nil {
		t.Fatal("expected provider from UPDATE_GITHUB_*")
	}
	if conf.SelfUpdate.CheckSchedule != "@every 12h" {
		t.Errorf("CheckSchedule = %q, want @every 12h", conf.SelfUpdate.CheckSchedule)
	}
	if !conf.SelfUpdate.AutoApply {
		t.Error("expected AutoApply from UPDATE_AUTO_APPLY")
	}
	if !conf.SelfUpdate.SystemdMode {
		t.Error("expected SystemdMode from UPDATE_SYSTEMD")
	}
}

func TestApplySelfUpdateEnvEnabledWithoutProviderFails(t *testing.T) {
	t.Setenv("UPDATE_ENABLED", "true")
	conf := DefaultConfig()
	if err := ApplyEnvOverrides(conf); err == nil {
		t.Fatal("expected error when UPDATE_ENABLED=true without a provider")
	}
}

func TestApplySelfUpdateEnvProgrammaticProviderWins(t *testing.T) {
	// A programmatically-configured SelfUpdate is never disabled by env; the
	// env provider only fills a nil Provider.
	conf := DefaultConfig()
	conf.SelfUpdate = &SelfUpdateConfig{}
	t.Setenv("UPDATE_ENABLED", "false")
	t.Setenv("UPDATE_GITHUB_OWNER", "acme")
	t.Setenv("UPDATE_GITHUB_REPO", "app")
	t.Setenv("UPDATE_GITHUB_ASSET_PATTERN", "app")
	if err := ApplyEnvOverrides(conf); err != nil {
		t.Fatalf("apply overrides: %v", err)
	}
	if conf.SelfUpdate.Provider == nil {
		t.Fatal("expected env provider to fill nil programmatic provider")
	}
}

// TestLoadConfigEnvFilePrecedence verifies the .env.dev > .env > process env
// precedence in LoadConfig.
func TestLoadConfigEnvFilePrecedence(t *testing.T) {
	dir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
		t.Setenv("PORT", "")
		t.Setenv("SMTP_PORT", "")
	})

	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "data"))
	t.Setenv("PORT", "1000")
	t.Setenv("SMTP_PORT", "25")

	if err := os.WriteFile(".env", []byte("PORT=1111\nSMTP_PORT=2526\n"), 0644); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	if err := os.WriteFile(".env.dev", []byte("PORT=2222\n"), 0644); err != nil {
		t.Fatalf("write .env.dev: %v", err)
	}

	cfg, err := LoadConfig("envprecedence")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Port != 2222 {
		t.Errorf("Port = %d, want 2222 (.env.dev beats .env and process env)", cfg.Port)
	}
	if cfg.Mailer.SMTPPort != 2526 {
		t.Errorf("SMTPPort = %d, want 2526 (.env beats process env when .env.dev is silent)", cfg.Mailer.SMTPPort)
	}
}
