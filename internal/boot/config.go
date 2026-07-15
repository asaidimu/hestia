package boot

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/asaidimu/hestia/internal/abstract"
	"github.com/asaidimu/hestia/internal/core"
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
		dataDir = filepath.Join(dataDir, "anansi")
	}
	_ = os.MkdirAll(dataDir, 0700)

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
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
		DBPath:          filepath.Join(dataDir, "anansi.db"),
		JWTSecret:       jwtSecret,
		BcryptCost:      envInt("BCRYPT_COST", core.DefaultBcryptCost),
		AccessTokenTTL:  envDuration("JWT_ACCESS_TTL", core.DefaultAccessTokenTTL),
		RefreshTokenTTL: envDuration("JWT_REFRESH_TTL", core.DefaultRefreshTokenTTL),
		ResetTokenTTL:   envDuration("JWT_RESET_TTL", core.DefaultResetTokenTTL),
		LogPath:         filepath.Join(dataDir, "server.log"),
		LogMaxSize:      100,
		LogMaxAge:       30,
		LogMaxBackups:   5,
		BlobsDir:        blobsDir,
		CookieConfig: core.CookieConfig{
			Domain:      os.Getenv("COOKIE_DOMAIN"),
			Secure:      envOrBool("COOKIE_SECURE", true),
			HTTPOnly:    true,
			SameSite:    parseSameSite(os.Getenv("COOKIE_SAMESITE")),
			AccessName:  envOrString("ACCESS_COOKIE_NAME", "access_token"),
			AccessPath:  envOrString("ACCESS_COOKIE_PATH", "/"),
			RefreshName: envOrString("REFRESH_COOKIE_NAME", "refresh_token"),
			RefreshPath: envOrString("REFRESH_COOKIE_PATH", "/api/auth/session"),
		},
	}, nil
}
