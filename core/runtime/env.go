package runtime

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"

	"github.com/asaidimu/updater"
)

// ApplyEnvOverrides re-applies every environment-backed knob on top of conf so
// env wins over hardcoded SetupConfig values (todo/general_todo.md). It is
// called by Setup after SetupConfig.applyTo. Non-empty env values override;
// absent values leave conf untouched. SelfUpdate is resolved from UPDATE_*
// env vars when it is not (or not fully) configured in code.
func ApplyEnvOverrides(conf *Config) error {
	if v := os.Getenv("APP_VERSION"); v != "" {
		conf.Version = v
	}
	if n, ok := envInt("PORT"); ok {
		conf.Port = n
	}
	if v := os.Getenv("SESSION_SECRET"); v != "" {
		conf.SessionSecret = v
	}
	if v := os.Getenv("DB_PATH"); v != "" {
		conf.DBPath = v
	}
	if v := os.Getenv("LOG_PATH"); v != "" {
		conf.LogPath = v
	}
	if v := os.Getenv("BLOBS_DIR"); v != "" {
		conf.BlobsDir = v
	}
	if n, ok := envInt("BCRYPT_COST"); ok {
		conf.BcryptCost = n
	}
	if d, ok := envDuration("SESSION_TTL"); ok {
		conf.SessionTTL = d
	}
	if d, ok := envDuration("SESSION_IDLE_TTL"); ok {
		conf.IdleTTL = d
	}
	if d, ok := envDuration("SESSION_REFRESH_TTL"); ok {
		conf.RefreshTTL = d
	}
	if v := os.Getenv("API_PREFIX"); v != "" {
		conf.APIPrefix = v
	}
	if v := os.Getenv("COOKIE_DOMAIN"); v != "" {
		conf.CookieConfig.Domain = v
	}
	if b, ok := envBool("COOKIE_SECURE"); ok {
		conf.CookieConfig.Secure = b
	}
	if v := os.Getenv("COOKIE_SAMESITE"); v != "" {
		conf.CookieConfig.SameSite = parseSameSite(v)
	}
	if v := os.Getenv("SESSION_COOKIE_NAME"); v != "" {
		conf.CookieConfig.SessionName = v
	}
	if v := os.Getenv("SESSION_COOKIE_PATH"); v != "" {
		conf.CookieConfig.SessionPath = v
	}
	if v := os.Getenv("ALLOWED_ORIGINS"); v != "" {
		parts := strings.Split(v, ",")
		conf.AllowedOrigins = make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				conf.AllowedOrigins = append(conf.AllowedOrigins, p)
			}
		}
	}
	if b, ok := envBool("FORCE_BOOTSTRAPPED"); ok {
		conf.ForceBootstrapped = b
	}
	if v := os.Getenv("SMTP_HOST"); v != "" {
		conf.Mailer.SMTPHost = v
	}
	if n, ok := envInt("SMTP_PORT"); ok {
		conf.Mailer.SMTPPort = n
	}
	if v := os.Getenv("SMTP_USERNAME"); v != "" {
		conf.Mailer.SMTPUsername = v
	}
	if v := os.Getenv("SMTP_PASSWORD"); v != "" {
		conf.Mailer.SMTPPassword = v
	}
	if v := os.Getenv("SMTP_FROM"); v != "" {
		conf.Mailer.FromAddress = v
	}
	if v := os.Getenv("SMTP_FROM_NAME"); v != "" {
		conf.Mailer.FromName = v
	}
	if v := os.Getenv("SMTP_AUTH_TYPE"); v != "" {
		conf.Mailer.SMTPAuthType = v
	}
	if v := os.Getenv("APP_URL"); v != "" {
		conf.AppURL = v
	}

	return applySelfUpdateEnv(conf)
}

// applySelfUpdateEnv builds or overlays SelfUpdate from UPDATE_* env vars.
// A programmatically-configured SelfUpdate is never disabled by env; env only
// fills a nil Provider and overrides scalar fields.
func applySelfUpdateEnv(conf *Config) error {
	su := conf.SelfUpdate
	if su == nil {
		if enabled, ok := envBool("UPDATE_ENABLED"); ok && !enabled {
			return nil
		}
		provider, err := providerFromEnv()
		if err != nil {
			return err
		}
		if provider == nil {
			if enabled, ok := envBool("UPDATE_ENABLED"); ok && enabled {
				return fmt.Errorf("UPDATE_ENABLED=true requires provider env vars (UPDATE_GITHUB_* or UPDATE_SERVER_*)")
			}
			return nil
		}
		su = &SelfUpdateConfig{Provider: provider}
		conf.SelfUpdate = su
	} else if su.Provider == nil {
		provider, err := providerFromEnv()
		if err != nil {
			return err
		}
		su.Provider = provider
	}

	if v := os.Getenv("UPDATE_CHECK_SCHEDULE"); v != "" {
		su.CheckSchedule = v
	}
	if v := os.Getenv("UPDATE_DATA_DIR"); v != "" {
		su.DataDir = v
	}
	if v := os.Getenv("UPDATE_EXECUTABLE_PATH"); v != "" {
		su.ExecutablePath = v
	}
	if b, ok := envBool("UPDATE_FORWARD_ARGUMENTS"); ok {
		su.ForwardArguments = b
	}
	if b, ok := envBool("UPDATE_AUTO_APPLY"); ok {
		su.AutoApply = b
	}
	if b, ok := envBool("UPDATE_SYSTEMD"); ok {
		su.SystemdMode = b
	}
	return nil
}

// providerFromEnv constructs an updater.Provider from env vars. It returns
// (nil, nil) when no provider env is present. Setting both provider families
// is an error.
func providerFromEnv() (updater.Provider, error) {
	gh := envSet("UPDATE_GITHUB_OWNER", "UPDATE_GITHUB_REPO", "UPDATE_GITHUB_ASSET_PATTERN", "UPDATE_GITHUB_TOKEN")
	srv := envSet("UPDATE_SERVER_URL", "UPDATE_SERVER_APP_NAME", "UPDATE_SERVER_CLIENT_TOKEN", "UPDATE_SERVER_CLIENT_ID", "UPDATE_SERVER_PUBLIC_KEY")

	if gh && srv {
		return nil, fmt.Errorf("both UPDATE_GITHUB_* and UPDATE_SERVER_* provider env vars are set; configure only one")
	}

	if gh {
		owner, repo, pattern := os.Getenv("UPDATE_GITHUB_OWNER"), os.Getenv("UPDATE_GITHUB_REPO"), os.Getenv("UPDATE_GITHUB_ASSET_PATTERN")
		if owner == "" || repo == "" || pattern == "" {
			return nil, fmt.Errorf("UPDATE_GITHUB_OWNER, UPDATE_GITHUB_REPO and UPDATE_GITHUB_ASSET_PATTERN are all required")
		}
		return updater.NewGitHubProvider(updater.GitHubConfig{
			Owner:        owner,
			Repo:         repo,
			Token:        os.Getenv("UPDATE_GITHUB_TOKEN"),
			AssetPattern: pattern,
		})
	}

	if srv {
		url, app, token, id := os.Getenv("UPDATE_SERVER_URL"), os.Getenv("UPDATE_SERVER_APP_NAME"), os.Getenv("UPDATE_SERVER_CLIENT_TOKEN"), os.Getenv("UPDATE_SERVER_CLIENT_ID")
		if url == "" || app == "" || token == "" || id == "" {
			return nil, fmt.Errorf("UPDATE_SERVER_URL, UPDATE_SERVER_APP_NAME, UPDATE_SERVER_CLIENT_TOKEN and UPDATE_SERVER_CLIENT_ID are all required")
		}
		key, err := parseRSAPublicKey(os.Getenv("UPDATE_SERVER_PUBLIC_KEY"))
		if err != nil {
			return nil, fmt.Errorf("UPDATE_SERVER_PUBLIC_KEY: %w", err)
		}
		return updater.NewServerProvider(updater.ServerConfig{
			ServerURL:       url,
			AppName:         app,
			ClientToken:     token,
			ServerPublicKey: key,
			ClientID:        id,
		})
	}

	return nil, nil
}

func envSet(keys ...string) bool {
	for _, k := range keys {
		if os.Getenv(k) != "" {
			return true
		}
	}
	return false
}

// parseRSAPublicKey decodes a PEM-encoded PKIX RSA public key.
func parseRSAPublicKey(raw string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil, fmt.Errorf("expected a PEM block")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	pub, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is %T, want *rsa.PublicKey", key)
	}
	return pub, nil
}