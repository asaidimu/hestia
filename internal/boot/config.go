package boot

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/asaidimu/hestia/app/abstract"
	"github.com/asaidimu/hestia/app/core"
)

func envOrString(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func envOrBool(key string, defaultVal bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return defaultVal
}

var ProjectName = "hestia"

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

func envDuration(key string, defaultVal time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultVal
}

func envInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}

func NewConfig() (*core.Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = ":8090"
	}

	projectName := ProjectName

	dataDir := os.Getenv("APP_DATA_DIR")
	if dataDir == "" {
		dataDir = os.Getenv("XDG_DATA_HOME")
		if dataDir == "" {
			home, err := os.UserHomeDir()
			if err == nil {
				dataDir = filepath.Join(home, ".local", "share")
			} else {
				dataDir = "./data"
			}
		}
		dataDir = filepath.Join(dataDir, projectName)
	}
	_ = os.MkdirAll(dataDir, 0700)

	signingSecret := os.Getenv("SESSION_SECRET")
	if signingSecret == "" {
		signingSecret = os.Getenv("JWT_SECRET")
	}
	if signingSecret == "" {
		return nil, fmt.Errorf("SESSION_SECRET environment variable is required")
	}

	blobsDir := os.Getenv("BLOBS_DIR")
	if blobsDir == "" {
		blobsDir = filepath.Join(dataDir, "blobs")
	}
	if err := os.MkdirAll(blobsDir, 0700); err != nil {
		return nil, fmt.Errorf("create blobs directory: %w", err)
	}

	return &core.Config{
		Port:            port,
		DataDir:         dataDir,
		DBPath:          filepath.Join(dataDir, projectName+".db"),
		SessionSecret:   signingSecret,
		BcryptCost:      envInt("BCRYPT_COST", core.DefaultBcryptCost),
		SessionTTL:      envDuration("SESSION_TTL", core.DefaultSessionTTL),
		IdleTTL:         envDuration("SESSION_IDLE_TTL", core.DefaultIdleTTL),
		RefreshTTL:      envDuration("SESSION_REFRESH_TTL", core.DefaultRefreshTTL),
		LogPath:         filepath.Join(dataDir, "server.log"),
		LogMaxSize:      100,
		LogMaxAge:       30,
		LogMaxBackups:   5,
		BlobsDir:        blobsDir,
		APIPrefix:       envOrString("API_PREFIX", "/api"),
		CookieConfig: core.CookieConfig{
			Domain:      os.Getenv("COOKIE_DOMAIN"),
			Secure:      envOrBool("COOKIE_SECURE", true),
			HTTPOnly:    true,
			SameSite:    parseSameSite(os.Getenv("COOKIE_SAMESITE")),
			SessionName: envOrString("SESSION_COOKIE_NAME", "session"),
			SessionPath: envOrString("SESSION_COOKIE_PATH", "/"),
		},
	}, nil
}
