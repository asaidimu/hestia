package core

import (
	"net/http"
	"time"

	"github.com/asaidimu/go-anansi/v8"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"go.uber.org/zap"
)

const (
	DefaultBcryptCost      = 12
	DefaultAccessTokenTTL  = 15 * time.Minute
	DefaultRefreshTokenTTL = 7 * 24 * time.Hour
	DefaultResetTokenTTL   = 5 * time.Minute
)

// InteractorFactory creates a database interactor and returns a cleanup function.
type InteractorFactory func(logger *zap.Logger) (query.DatabaseInteractor, func(), error)

type Config struct {
	Port          string
	DataDir       string
	DBPath        string
	JWTSecret     string
	LogPath       string
	LogMaxSize    int
	LogMaxAge     int
	LogMaxBackups int
	BlobsDir      string

	// BcryptCost is the cost factor for bcrypt password hashing.
	// Defaults to DefaultBcryptCost (12).
	BcryptCost int

	// AccessTokenTTL, RefreshTokenTTL, ResetTokenTTL control JWT token lifetimes.
	// Defaults to DefaultAccessTokenTTL (15m), DefaultRefreshTokenTTL (168h), DefaultResetTokenTTL (5m).
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	ResetTokenTTL   time.Duration

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
	// SameSite controls CSRF protection (default http.SameSiteStrictMode).
	SameSite http.SameSite

	// AccessName is the access token cookie name (default "access_token").
	AccessName string
	// AccessPath restricts the access cookie to a path (default "/").
	AccessPath string

	// RefreshName is the refresh token cookie name (default "refresh_token").
	RefreshName string
	// RefreshPath restricts the refresh cookie to a path (default "/api/auth/session").
	RefreshPath string
}
