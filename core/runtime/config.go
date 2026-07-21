package runtime

import (
	"io/fs"
	"time"

	"github.com/asaidimu/go-anansi/v8"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"github.com/asaidimu/hestia/core/abstract"
	"go.uber.org/zap"
)

const (
	DefaultBcryptCost = 12

	// Session defaults (sliding window).
	DefaultSessionTTL  = 8 * time.Hour // Absolute session lifetime (1 work day)
	DefaultIdleTTL     = 30 * time.Minute    // Max idle time before session expires
	DefaultRefreshTTL  = 15 * time.Minute    // Refresh session cookie after this idle period
	DefaultResetTTL    = 5 * time.Minute     // Password reset token lifetime
)

// InteractorFactory creates a database interactor and returns a cleanup function.
type InteractorFactory func(logger *zap.Logger) (query.DatabaseInteractor, func(), error)

type Config struct {
	Port          string
	DataDir       string
	DBPath        string
	SessionSecret string
	LogPath       string
	LogMaxSize    int
	LogMaxAge     int
	LogMaxBackups int
	BlobsDir      string

	// BcryptCost is the cost factor for bcrypt password hashing.
	// Defaults to DefaultBcryptCost (12).
	BcryptCost int

	// SessionTTL is the absolute session lifetime (default 7 days).
	SessionTTL time.Duration
	// IdleTTL is the maximum idle time before a session expires (default 30 min).
	IdleTTL time.Duration
	// RefreshTTL is the idle threshold after which the session cookie is refreshed (default 15 min).
	RefreshTTL time.Duration

	// InteractorFactory overrides the default SQLite database creation.
	// When set, the factory is called to create the database interactor.
	InteractorFactory InteractorFactory

	// PersistenceFactory gives full control over persistence setup.
	// Receives a minimal *anansi.SetupConfig (logger + sanitization rules),
	// returns a base.Persistence. No interactor, event bus, or database
	// is created beforehand — you own the full lifecycle.
	PersistenceFactory func(cfg *anansi.SetupConfig) (base.Persistence, error)

	// AdminEmail and AdminPassword override the random seed admin credentials.
	AdminEmail    string
	AdminPassword string

	// ForceBootstrapped skips the bootstrap flow and marks the system as bootstrapped.
	ForceBootstrapped bool

	// APIPrefix is the URL prefix for all API routes (e.g. "/api").
	// Empty string means no prefix.
	APIPrefix string

	// StaticFS serves static files for the SPA at the root path.
	// When set, unmatched non-API routes fall through to file serving
	// with index.html fallback for client-side routing.
	StaticFS fs.FS

	// CookieConfig controls httpOnly cookie settings for token cookies.
	CookieConfig CookieConfig
}

type CookieConfig struct {
	// Domain restricts cookies to a specific domain.
	Domain string
	// Secure requires HTTPS (default true).
	Secure bool
	// HTTPOnly prevents JavaScript access (default true).
	HTTPOnly bool
	// SameSite controls CSRF protection (default abstract.SameSiteStrictMode).
	SameSite abstract.SameSite

	// SessionName is the session cookie name (default "session").
	SessionName string
	// SessionPath restricts the session cookie to a path (default "/").
	SessionPath string
}
