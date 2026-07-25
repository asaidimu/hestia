package http

import (
	"time"

	"github.com/asaidimu/hestia/core/runtime"
	"github.com/valyala/fasthttp"
)

func toFasthttpCookie(c Cookie) *fasthttp.Cookie {
	fc := fasthttp.Cookie{}
	fc.SetKey(c.Name)
	fc.SetValue(c.Value)
	fc.SetPath(c.Path)
	fc.SetDomain(c.Domain)
	fc.SetMaxAge(c.MaxAge)
	fc.SetSecure(c.Secure)
	fc.SetHTTPOnly(c.HTTPOnly)
	fc.SetSameSite(mapSameSite(c.SameSite))
	return &fc
}

func newSessionCookie(name, value, path string, ttl time.Duration, cfg runtime.CookieConfig) Cookie {
	return Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Domain:   cfg.Domain,
		Secure:   cfg.Secure,
		HTTPOnly: cfg.HTTPOnly,
		SameSite: cfg.SameSite,
		MaxAge:   int(ttl.Seconds()),
	}
}

func clearSessionCookie(name, path string, cfg runtime.CookieConfig) Cookie {
	return Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		Domain:   cfg.Domain,
		Secure:   cfg.Secure,
		HTTPOnly: cfg.HTTPOnly,
		SameSite: cfg.SameSite,
		MaxAge:   0,
	}
}
