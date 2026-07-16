package boot

import (
	"os"
	"testing"
	"time"

	"github.com/asaidimu/hestia/app/abstract"
	"github.com/asaidimu/hestia/app/core"
)

func TestNewConfigDefaults(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
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
	if cfg.JWTSecret != "test-secret" {
		t.Errorf("JWTSecret = %q, want %q", cfg.JWTSecret, "test-secret")
	}
	if cfg.BcryptCost != core.DefaultBcryptCost {
		t.Errorf("BcryptCost = %d, want %d", cfg.BcryptCost, core.DefaultBcryptCost)
	}
	if cfg.AccessTokenTTL != core.DefaultAccessTokenTTL {
		t.Errorf("AccessTokenTTL = %v, want %v", cfg.AccessTokenTTL, core.DefaultAccessTokenTTL)
	}
	if cfg.RefreshTokenTTL != core.DefaultRefreshTokenTTL {
		t.Errorf("RefreshTokenTTL = %v, want %v", cfg.RefreshTokenTTL, core.DefaultRefreshTokenTTL)
	}
	if cfg.ResetTokenTTL != core.DefaultResetTokenTTL {
		t.Errorf("ResetTokenTTL = %v, want %v", cfg.ResetTokenTTL, core.DefaultResetTokenTTL)
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
	if cc.AccessName != "access_token" {
		t.Errorf("CookieConfig.AccessName = %q, want %q", cc.AccessName, "access_token")
	}
	if cc.AccessPath != "/" {
		t.Errorf("CookieConfig.AccessPath = %q, want %q", cc.AccessPath, "/")
	}
	if cc.RefreshName != "refresh_token" {
		t.Errorf("CookieConfig.RefreshName = %q, want %q", cc.RefreshName, "refresh_token")
	}
	if cc.RefreshPath != "/api/auth/session" {
		t.Errorf("CookieConfig.RefreshPath = %q, want %q", cc.RefreshPath, "/api/auth/session")
	}
}

func TestNewConfigCustom(t *testing.T) {
	t.Setenv("PORT", ":9999")
	t.Setenv("JWT_SECRET", "custom-secret")
	t.Setenv("APP_DATA_DIR", "/tmp/hestia-custom/data")
	t.Setenv("BLOBS_DIR", "/tmp/hestia-custom/blobs")
	t.Setenv("BCRYPT_COST", "10")
	t.Setenv("JWT_ACCESS_TTL", "30m")
	t.Setenv("JWT_REFRESH_TTL", "336h")
	t.Setenv("JWT_RESET_TTL", "10m")
	t.Setenv("COOKIE_DOMAIN", "example.com")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("COOKIE_SAMESITE", "lax")
	t.Setenv("ACCESS_COOKIE_NAME", "at")
	t.Setenv("ACCESS_COOKIE_PATH", "/app")
	t.Setenv("REFRESH_COOKIE_NAME", "rt")
	t.Setenv("REFRESH_COOKIE_PATH", "/auth")

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
	if cfg.JWTSecret != "custom-secret" {
		t.Errorf("JWTSecret = %q, want %q", cfg.JWTSecret, "custom-secret")
	}
	if cfg.BcryptCost != 10 {
		t.Errorf("BcryptCost = %d, want 10", cfg.BcryptCost)
	}
	if cfg.AccessTokenTTL != 30*time.Minute {
		t.Errorf("AccessTokenTTL = %v, want 30m", cfg.AccessTokenTTL)
	}
	if cfg.RefreshTokenTTL != 336*time.Hour {
		t.Errorf("RefreshTokenTTL = %v, want 336h", cfg.RefreshTokenTTL)
	}
	if cfg.ResetTokenTTL != 10*time.Minute {
		t.Errorf("ResetTokenTTL = %v, want 10m", cfg.ResetTokenTTL)
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
	if cc.AccessName != "at" {
		t.Errorf("CookieConfig.AccessName = %q, want %q", cc.AccessName, "at")
	}
	if cc.AccessPath != "/app" {
		t.Errorf("CookieConfig.AccessPath = %q, want %q", cc.AccessPath, "/app")
	}
	if cc.RefreshName != "rt" {
		t.Errorf("CookieConfig.RefreshName = %q, want %q", cc.RefreshName, "rt")
	}
	if cc.RefreshPath != "/auth" {
		t.Errorf("CookieConfig.RefreshPath = %q, want %q", cc.RefreshPath, "/auth")
	}
}

func TestNewConfigMissingJWTSecret(t *testing.T) {
	os.Unsetenv("JWT_SECRET")
	_, err := NewConfig()
	if err == nil {
		t.Fatal("NewConfig() expected error when JWT_SECRET is unset")
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
	cfg := &core.Config{
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
