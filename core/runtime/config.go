// @note #org-20260821-003 todo status=open priority=P2 tags=#organization,#refactor : Split config.go into types.go, defaults.go, and env.go
//
// config.go currently contains 3 unrelated concerns in 321 lines:
// - Config, SelfUpdateConfig, CookieConfig types (data structures)
// - Default constants and DefaultConfig() (defaults)
// - LoadConfig, env helpers, resolveDataDir (env loading)
//
// A human looking for Config struct would guess config.go (correct), but
// a human looking for env helpers would guess env.go (which exists but
// only has ApplyEnvOverrides). The file mixes type definitions with
// runtime behavior.
//
// Resolution:
// 1. Create types.go with Config, SelfUpdateConfig, CookieConfig structs
// 2. Create defaults.go with Default* constants and DefaultConfig()
// 3. Keep config.go with LoadConfig, env helpers, resolveDataDir
//    (or rename to load.go for clarity)
//
// This also addresses #so98l5 and #r9i7ec by consolidating env resolution.
//
// Files affected:
// - core/runtime/config.go (split into 3 files)
// - core/runtime/env.go (consider merging ApplyEnvOverrides into config.go)
// - core/hestia.go (imports Config)
package runtime

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/asaidimu/go-anansi/v8"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"github.com/asaidimu/updater"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
)

// @note #so98l5 todo resolved : Define a proper config package.
// Consider the state of this collection of adhoc helpers
// that are duplicated everywhere. It would be best to define
// a single package that holds all configs, arbitary or otherwise,
// backed by a schema generated from a struct. Anansi is not just a
// data layer but a data contract.

const (
	DefaultPort          = 8090
	DefaultBcryptCost    = 12
	DefaultLogMaxSize    = 100
	DefaultLogMaxAge     = 30
	DefaultLogMaxBackups = 5

	DefaultSessionTTL = 8 * time.Hour
	DefaultIdleTTL    = 30 * time.Minute
	DefaultRefreshTTL = 15 * time.Minute
	DefaultResetTTL   = 5 * time.Minute

	DefaultAPIPrefix     = "/api"
	DefaultSessionCookie = "session"
	DefaultSessionSecret = "3ecb5a2ef5014f88-8a00-8227db8b7298"
	DefaultSessionPath   = "/"
)

type InteractorFactory func(logger *zap.Logger) (query.DatabaseInteractor, func(), error)

type Config struct {
	Port          int
	DataDir       string
	DBPath        string
	SessionSecret string
	LogPath       string
	LogMaxSize    int
	LogMaxAge     int
	LogMaxBackups int
	BlobsDir      string
	BcryptCost    int
	SessionTTL    time.Duration
	IdleTTL       time.Duration
	RefreshTTL    time.Duration

	Version    string
	SelfUpdate *SelfUpdateConfig

	InteractorFactory  InteractorFactory
	PersistenceFactory func(cfg *anansi.SetupConfig) (base.Persistence, error)

	AdminEmail        string
	AdminPassword     string
	ForceBootstrapped bool
	APIPrefix         string
	StaticFS          fs.FS
	CookieConfig      CookieConfig
	AllowedOrigins    []string
	Mailer            MailerConfig
	AppURL            string
}

// SelfUpdateConfig enables hestia's built-in self-update service. It lives in
// runtime (a leaf package) because both core/hestia.go and the updates service
// need it, and neither can import the service without a cycle.
type SelfUpdateConfig struct {
	// Provider is where updates come from: updater.NewGitHubProvider,
	// updater.NewServerProvider, or a custom updater.Provider. When nil, the
	// provider is resolved from UPDATE_GITHUB_* / UPDATE_SERVER_* env vars.
	Provider updater.Provider

	// CheckSchedule is a cron expression for the recurring check; empty
	// disables the scheduled job (manual checks still work). Default "@every 24h".
	CheckSchedule string

	// DataDir is where staged update binaries live. Defaults to conf.DataDir.
	DataDir string
	// ExecutablePath is where the new binary is copied on swap. Defaults to os.Executable().
	ExecutablePath string

	// ForwardArguments restores the original CLI arguments after the swap.
	ForwardArguments bool
	// AutoApply applies an available update automatically during the scheduled
	// check. Default false — ApplyUpdate is always the explicit admin action.
	AutoApply bool

	// SystemdMode deploys the process as a systemd-managed daemon. Instead of
	// the spawn-and-swap handoff (ApplyUpdate spawning a --perform-update
	// child that replaces the executable), apply copies the staged binary over
	// the executable in place and then exits cleanly, letting systemd's
	// Restart=always launch the new binary as a tracked unit. Requires the
	// executable path to be writable by the service user.
	SystemdMode bool
}

type CookieConfig struct {
	Domain      string
	Secure      bool
	HTTPOnly    bool
	SameSite    abstract.SameSite
	SessionName string
	SessionPath string
}

func DefaultConfig() *Config {
	return &Config{
		Port:           DefaultPort,
		BcryptCost:     DefaultBcryptCost,
		SessionTTL:     DefaultSessionTTL,
		SessionSecret:  DefaultSessionSecret,
		IdleTTL:        DefaultIdleTTL,
		RefreshTTL:     DefaultRefreshTTL,
		LogMaxSize:     DefaultLogMaxSize,
		LogMaxAge:      DefaultLogMaxAge,
		LogMaxBackups:  DefaultLogMaxBackups,
		APIPrefix:      DefaultAPIPrefix,
		AllowedOrigins: []string{},
		CookieConfig: CookieConfig{
			Secure:      true,
			HTTPOnly:    true,
			SameSite:    abstract.SameSiteStrictMode,
			SessionName: DefaultSessionCookie,
			SessionPath: DefaultSessionPath,
		},
	}
}

func resolveDataDir(projectName string) string {
	if d := os.Getenv("APP_DATA_DIR"); d != "" {
		return d
	}
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, projectName)
	}
	home, err := os.UserHomeDir()
	if err == nil {
		d := filepath.Join(home, ".local", "share", projectName)
		_ = os.MkdirAll(d, 0700)
		return d
	}
	return "./data"
}

func envString(key string) (string, bool) {
	if v := os.Getenv(key); v != "" {
		return v, true
	}
	return "", false
}

func envInt(key string) (int, bool) {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n, true
		}
	}
	return 0, false
}

func envDuration(key string) (time.Duration, bool) {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d, true
		}
	}
	return 0, false
}

func envBool(key string) (bool, bool) {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b, true
		}
	}
	return false, false
}

func LoadConfig(projectName string) (*Config, error) {
	// Precedence (lowest wins last): process env < .env < .env.dev. Overload
	// (not Load) so each file overrides the layer below it.
	_ = godotenv.Overload()
	_ = godotenv.Overload(".env.dev")

	cfg := DefaultConfig()

	if projectName == "" {
		projectName = "hestia"
	}

	cfg.DataDir = resolveDataDir(projectName)
	_ = os.MkdirAll(cfg.DataDir, 0700)

	cfg.DBPath = filepath.Join(cfg.DataDir, projectName+".db")
	cfg.LogPath = filepath.Join(cfg.DataDir, "server.log")

	blobsDir := os.Getenv("BLOBS_DIR")
	if blobsDir == "" {
		blobsDir = filepath.Join(cfg.DataDir, "blobs")
	}
	cfg.BlobsDir = blobsDir
	_ = os.MkdirAll(cfg.BlobsDir, 0700)

	applyCommonEnvOverrides(cfg)

	return cfg, nil
}

// applyCommonEnvOverrides applies environment-backed knobs that are shared
// between LoadConfig (startup) and ApplyEnvOverrides (Setup). Non-empty env
// values override; absent values leave conf untouched.
func applyCommonEnvOverrides(cfg *Config) {
	if n, ok := envInt("PORT"); ok {
		cfg.Port = n
	}
	if n, ok := envInt("BCRYPT_COST"); ok {
		cfg.BcryptCost = n
	}
	if d, ok := envDuration("SESSION_TTL"); ok {
		cfg.SessionTTL = d
	}
	if d, ok := envDuration("SESSION_IDLE_TTL"); ok {
		cfg.IdleTTL = d
	}
	if d, ok := envDuration("SESSION_REFRESH_TTL"); ok {
		cfg.RefreshTTL = d
	}
	if v := os.Getenv("API_PREFIX"); v != "" {
		cfg.APIPrefix = v
	}
	if v := os.Getenv("COOKIE_DOMAIN"); v != "" {
		cfg.CookieConfig.Domain = v
	}
	if b, ok := envBool("COOKIE_SECURE"); ok {
		cfg.CookieConfig.Secure = b
	}
	if v := os.Getenv("COOKIE_SAMESITE"); v != "" {
		cfg.CookieConfig.SameSite = parseSameSite(v)
	}
	if v := os.Getenv("SESSION_COOKIE_NAME"); v != "" {
		cfg.CookieConfig.SessionName = v
	}
	if v := os.Getenv("SESSION_COOKIE_PATH"); v != "" {
		cfg.CookieConfig.SessionPath = v
	}
	if secret, ok := envString("SESSION_SECRET"); ok {
		cfg.SessionSecret = secret
	}
	if v := os.Getenv("DB_PATH"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("LOG_PATH"); v != "" {
		cfg.LogPath = v
	}
	if v := os.Getenv("BLOBS_DIR"); v != "" {
		cfg.BlobsDir = v
	}
	if v := os.Getenv("ALLOWED_ORIGINS"); v != "" {
		parts := strings.Split(v, ",")
		cfg.AllowedOrigins = make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				cfg.AllowedOrigins = append(cfg.AllowedOrigins, p)
			}
		}
	}
	if b, ok := envBool("FORCE_BOOTSTRAPPED"); ok {
		cfg.ForceBootstrapped = b
	}
	if v := os.Getenv("SMTP_HOST"); v != "" {
		cfg.Mailer.SMTPHost = v
	}
	if n, ok := envInt("SMTP_PORT"); ok {
		cfg.Mailer.SMTPPort = n
	}
	if v := os.Getenv("SMTP_USERNAME"); v != "" {
		cfg.Mailer.SMTPUsername = v
	}
	if v := os.Getenv("SMTP_PASSWORD"); v != "" {
		cfg.Mailer.SMTPPassword = v
	}
	if v := os.Getenv("SMTP_FROM"); v != "" {
		cfg.Mailer.FromAddress = v
	}
	if v := os.Getenv("SMTP_FROM_NAME"); v != "" {
		cfg.Mailer.FromName = v
	}
	if v := os.Getenv("SMTP_AUTH_TYPE"); v != "" {
		cfg.Mailer.SMTPAuthType = v
	}
	if v := os.Getenv("APP_URL"); v != "" {
		cfg.AppURL = v
	}
	if v := os.Getenv("APP_VERSION"); v != "" {
		cfg.Version = v
	}
}

func parseSameSite(s string) abstract.SameSite {
	switch strings.ToLower(s) {
	case "lax":
		return abstract.SameSiteLaxMode
	case "none":
		return abstract.SameSiteNoneMode
	default:
		return abstract.SameSiteStrictMode
	}
}
