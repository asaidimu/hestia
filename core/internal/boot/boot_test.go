package boot

import (
	"os"
	"testing"
	"time"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
)

func TestNewConfigDefaults(t *testing.T) {
	t.Setenv("SESSION_SECRET", "test-secret")
	t.Setenv("APP_DATA_DIR", "/tmp/hestia-test-defaults")

	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() error: %v", err)
	}

	if cfg.Port != ":8090" {
		t.Errorf("Port = %q, want %q", cfg.Port, ":8090")
	}
	if cfg.DataDir != "/tmp/hestia-test-defaults" {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, "/tmp/hestia-test-defaults")
	}
	if cfg.DBPath != "/tmp/hestia-test-defaults/hestia.db" {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, "/tmp/hestia-test-defaults/hestia.db")
	}
	if cfg.LogPath != "/tmp/hestia-test-defaults/server.log" {
		t.Errorf("LogPath = %q, want %q", cfg.LogPath, "/tmp/hestia-test-defaults/server.log")
	}
	if cfg.BlobsDir != "/tmp/hestia-test-defaults/blobs" {
		t.Errorf("BlobsDir = %q, want %q", cfg.BlobsDir, "/tmp/hestia-test-defaults/blobs")
	}
	if cfg.SessionSecret != "test-secret" {
		t.Errorf("SessionSecret = %q, want %q", cfg.SessionSecret, "test-secret")
	}
	if cfg.BcryptCost != runtime.DefaultBcryptCost {
		t.Errorf("BcryptCost = %d, want %d", cfg.BcryptCost, runtime.DefaultBcryptCost)
	}
	if cfg.SessionTTL != runtime.DefaultSessionTTL {
		t.Errorf("SessionTTL = %v, want %v", cfg.SessionTTL, runtime.DefaultSessionTTL)
	}
	if cfg.IdleTTL != runtime.DefaultIdleTTL {
		t.Errorf("IdleTTL = %v, want %v", cfg.IdleTTL, runtime.DefaultIdleTTL)
	}
	if cfg.RefreshTTL != runtime.DefaultRefreshTTL {
		t.Errorf("RefreshTTL = %v, want %v", cfg.RefreshTTL, runtime.DefaultRefreshTTL)
	}

	cc := cfg.CookieConfig
	if !cc.Secure {
		t.Error("CookieConfig.Secure = false, want true")
	}
	if !cc.HTTPOnly {
		t.Error("CookieConfig.HTTPOnly = false, want true")
	}
	if cc.SameSite != abstract.SameSiteStrictMode {
		t.Errorf("CookieConfig.SameSite = %v, want SameSiteStrictMode", cc.SameSite)
	}
	if cc.SessionName != "session" {
		t.Errorf("CookieConfig.SessionName = %q, want %q", cc.SessionName, "session")
	}
	if cc.SessionPath != "/" {
		t.Errorf("CookieConfig.SessionPath = %q, want %q", cc.SessionPath, "/")
	}
}

func TestNewConfigCustom(t *testing.T) {
	t.Setenv("PORT", ":9999")
	t.Setenv("SESSION_SECRET", "custom-secret")
	t.Setenv("APP_DATA_DIR", "/tmp/hestia-custom/data")
	t.Setenv("BLOBS_DIR", "/tmp/hestia-custom/blobs")
	t.Setenv("BCRYPT_COST", "10")
	t.Setenv("SESSION_TTL", "168h")
	t.Setenv("SESSION_IDLE_TTL", "1h")
	t.Setenv("SESSION_REFRESH_TTL", "30m")
	t.Setenv("COOKIE_DOMAIN", "example.com")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("COOKIE_SAMESITE", "lax")
	t.Setenv("SESSION_COOKIE_NAME", "sid")
	t.Setenv("SESSION_COOKIE_PATH", "/app")

	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() error: %v", err)
	}

	if cfg.Port != ":9999" {
		t.Errorf("Port = %q, want %q", cfg.Port, ":9999")
	}
	if cfg.DataDir != "/tmp/hestia-custom/data" {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, "/tmp/hestia-custom/data")
	}
	if cfg.DBPath != "/tmp/hestia-custom/data/hestia.db" {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, "/tmp/hestia-custom/data/hestia.db")
	}
	if cfg.LogPath != "/tmp/hestia-custom/data/server.log" {
		t.Errorf("LogPath = %q, want %q", cfg.LogPath, "/tmp/hestia-custom/data/server.log")
	}
	if cfg.BlobsDir != "/tmp/hestia-custom/blobs" {
		t.Errorf("BlobsDir = %q, want %q", cfg.BlobsDir, "/tmp/hestia-custom/blobs")
	}
	if cfg.SessionSecret != "custom-secret" {
		t.Errorf("SessionSecret = %q, want %q", cfg.SessionSecret, "custom-secret")
	}
	if cfg.BcryptCost != 10 {
		t.Errorf("BcryptCost = %d, want 10", cfg.BcryptCost)
	}
	if cfg.SessionTTL != 168*time.Hour {
		t.Errorf("SessionTTL = %v, want 168h", cfg.SessionTTL)
	}
	if cfg.IdleTTL != 1*time.Hour {
		t.Errorf("IdleTTL = %v, want 1h", cfg.IdleTTL)
	}
	if cfg.RefreshTTL != 30*time.Minute {
		t.Errorf("RefreshTTL = %v, want 30m", cfg.RefreshTTL)
	}

	cc := cfg.CookieConfig
	if cc.Domain != "example.com" {
		t.Errorf("CookieConfig.Domain = %q, want %q", cc.Domain, "example.com")
	}
	if cc.Secure {
		t.Error("CookieConfig.Secure = true, want false")
	}
	if cc.SameSite != abstract.SameSiteLaxMode {
		t.Errorf("CookieConfig.SameSite = %v, want SameSiteLaxMode", cc.SameSite)
	}
	if cc.SessionName != "sid" {
		t.Errorf("CookieConfig.SessionName = %q, want %q", cc.SessionName, "sid")
	}
	if cc.SessionPath != "/app" {
		t.Errorf("CookieConfig.SessionPath = %q, want %q", cc.SessionPath, "/app")
	}
}

func TestNewConfigMissingSessionSecret(t *testing.T) {
	os.Unsetenv("SESSION_SECRET")
	os.Unsetenv("JWT_SECRET")
	_, err := NewConfig()
	if err == nil {
		t.Fatal("NewConfig() expected error when SESSION_SECRET is unset")
	}
}

func TestNewConfigFallsBackToJWTSecret(t *testing.T) {
	os.Unsetenv("SESSION_SECRET")
	t.Setenv("JWT_SECRET", "fallback-secret")
	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("NewConfig() error with JWT_SECRET fallback: %v", err)
	}
	if cfg.SessionSecret != "fallback-secret" {
		t.Errorf("SessionSecret = %q, want %q", cfg.SessionSecret, "fallback-secret")
	}
}

func TestEnvOrString(t *testing.T) {
	t.Setenv("TEST_STR_KEY", "hello")
	if got := envOrString("TEST_STR_KEY", "default"); got != "hello" {
		t.Errorf("envOrString = %q, want %q", got, "hello")
	}

	os.Unsetenv("TEST_STR_KEY_2")
	if got := envOrString("TEST_STR_KEY_2", "fallback"); got != "fallback" {
		t.Errorf("envOrString = %q, want %q", got, "fallback")
	}
}

func TestEnvOrBool(t *testing.T) {
	t.Setenv("TEST_BOOL_TRUE", "true")
	if got := envOrBool("TEST_BOOL_TRUE", false); got != true {
		t.Error("envOrBool(\"true\") = false, want true")
	}

	t.Setenv("TEST_BOOL_FALSE", "false")
	if got := envOrBool("TEST_BOOL_FALSE", true); got != false {
		t.Error("envOrBool(\"false\") = true, want false")
	}

	t.Setenv("TEST_BOOL_1", "1")
	if got := envOrBool("TEST_BOOL_1", false); got != true {
		t.Error("envOrBool(\"1\") = false, want true")
	}

	t.Setenv("TEST_BOOL_0", "0")
	if got := envOrBool("TEST_BOOL_0", true); got != false {
		t.Error("envOrBool(\"0\") = true, want false")
	}

	os.Unsetenv("TEST_BOOL_DEFAULT")
	if got := envOrBool("TEST_BOOL_DEFAULT", true); got != true {
		t.Error("envOrBool(unset) = false, want true")
	}
}

func TestEnvDuration(t *testing.T) {
	t.Setenv("TEST_DUR", "5m")
	if got := envDuration("TEST_DUR", 0); got != 5*time.Minute {
		t.Errorf("envDuration = %v, want 5m", got)
	}

	t.Setenv("TEST_DUR_HOURS", "2h")
	if got := envDuration("TEST_DUR_HOURS", 0); got != 2*time.Hour {
		t.Errorf("envDuration = %v, want 2h", got)
	}

	os.Unsetenv("TEST_DUR_DEFAULT")
	if got := envDuration("TEST_DUR_DEFAULT", 30*time.Second); got != 30*time.Second {
		t.Errorf("envDuration = %v, want 30s", got)
	}
}

func TestNewLoggers(t *testing.T) {
	cfg := &runtime.Config{
		LogPath:       t.TempDir() + "/hestia-test.log",
		LogMaxSize:    100,
		LogMaxAge:     30,
		LogMaxBackups: 5,
	}

	loggers := NewLoggers(cfg)
	if loggers == nil {
		t.Fatal("NewLoggers() returned nil")
	}
	if loggers.File == nil {
		t.Error("Loggers.File is nil")
	}
	if loggers.Stdout == nil {
		t.Error("Loggers.Stdout is nil")
	}
	loggers.Close()
}

func TestNewUserOutput(t *testing.T) {
	u := NewUserOutput()
	if u == nil {
		t.Fatal("NewUserOutput() returned nil")
	}
}
