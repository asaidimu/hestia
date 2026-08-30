package runtime

// Session secret provisioning and purpose-key derivation.
//
// Session tokens and password-reset tokens are both HMAC-SHA256 signed with
// keys derived from Config.SessionSecret. A secret that ships as a public
// constant makes every deployment forgeable, so there is no default: the
// secret must come from the operator (SESSION_SECRET, SetupConfig, or
// Config.SessionSecret) or be generated on first boot and persisted in the
// data directory with 0600 permissions. Boot must refuse to start when no
// secret can be provisioned.

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/hkdf"
)

// EnsureSessionSecret resolves cfg.SessionSecret.
//
//   - An explicitly configured secret (env, .env, SetupConfig, or code) wins
//     and is left untouched.
//   - Otherwise the secret is loaded from <DataDir>/session.key (0600).
//   - On first boot a fresh 32-byte random secret is generated, persisted
//     with 0600 permissions, and set on the config.
//
// It returns an error when no secret can be provisioned; callers must refuse
// to start rather than fall back to a known constant or an empty key.
func EnsureSessionSecret(cfg *Config) error {
	if strings.TrimSpace(cfg.SessionSecret) != "" {
		return nil
	}
	if cfg.DataDir == "" {
		return fmt.Errorf("session secret: no SESSION_SECRET configured and no data dir to persist one in")
	}
	path := filepath.Join(cfg.DataDir, sessionSecretFileName)

	if b, err := os.ReadFile(path); err == nil {
		if secret := strings.TrimSpace(string(b)); secret != "" {
			cfg.SessionSecret = secret
			return nil
		}
		// An empty key file is treated as absent and regenerated below.
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("session secret: read %s: %w", path, err)
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("session secret: generate: %w", err)
	}
	secret := hex.EncodeToString(key)
	if err := os.WriteFile(path, []byte(secret+"\n"), 0o600); err != nil {
		return fmt.Errorf("session secret: persist %s: %w", path, err)
	}
	cfg.SessionSecret = secret
	return nil
}

// hkdfSalt namespaces hestia's derivations so the same operator secret reused
// across tools does not collide with other HKDF uses.
const hkdfSalt = "hestia/v1/session-keys"

// DerivePurposeKey derives an independent, purpose-bound key from the master
// session secret using HKDF-SHA256 (returned hex-encoded).
//
// Purposes in use: "session" (session-token MAC) and "password-reset" (reset
// token MAC). Deriving per-purpose keys instead of concatenating a suffix
// ("secret" vs "secret:reset") keeps the two MAC keys cryptographically
// independent, so a compromise or reuse of one cannot weaken the other.
func DerivePurposeKey(masterSecret, purpose string) (string, error) {
	if strings.TrimSpace(masterSecret) == "" {
		return "", fmt.Errorf("derive %s key: master session secret is empty (is SESSION_SECRET set or provisioned?)", purpose)
	}
	r := hkdf.New(sha256.New, []byte(masterSecret), []byte(hkdfSalt), []byte(purpose))
	key := make([]byte, 32)
	if _, err := io.ReadFull(r, key); err != nil {
		return "", fmt.Errorf("derive %s key: %w", purpose, err)
	}
	return hex.EncodeToString(key), nil
}
