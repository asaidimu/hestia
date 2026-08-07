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
	"github.com/asaidimu/hestia/core/abstract"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

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
	_ = godotenv.Load()
	_ = godotenv.Load(".env.dev")

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

	cfg.SessionSecret = os.Getenv("SESSION_SECRET")

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
	if v := os.Getenv("DB_PATH"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("LOG_PATH"); v != "" {
		cfg.LogPath = v
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

	return cfg, nil
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
