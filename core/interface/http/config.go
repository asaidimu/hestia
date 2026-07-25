package http

import (
	"fmt"
	"io/fs"
	"time"

	"github.com/asaidimu/hestia/core/runtime"
)

type Config struct {
	Port              int
	APIPrefix         string
	StaticFS          fs.FS
	Middleware        []Middleware
	CookieConfig      runtime.CookieConfig
	SessionTTL        time.Duration
	IdleTTL           time.Duration
	RefreshTTL        time.Duration
	NoRefreshCommands []string
	AllowedOrigins    []string
}

func (c Config) Addr() string {
	port := c.Port
	if port <= 0 {
		port = runtime.DefaultPort
	}
	return fmt.Sprintf(":%d", port)
}

func ConfigFromRuntime(cfg *runtime.Config) Config {
	return Config{
		Port:              cfg.Port,
		APIPrefix:         cfg.APIPrefix,
		StaticFS:          cfg.StaticFS,
		CookieConfig:      cfg.CookieConfig,
		SessionTTL:        cfg.SessionTTL,
		IdleTTL:           cfg.IdleTTL,
		RefreshTTL:        cfg.RefreshTTL,
		AllowedOrigins:    cfg.AllowedOrigins,
	}
}
