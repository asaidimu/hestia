package notification

import (
	"fmt"
	"net/url"

	"github.com/asaidimu/hestia/core/abstract"
)

type templateFunc func(data map[string]any) string

var subjects = map[string]templateFunc{}
var bodies = map[string]map[abstract.ChannelType]templateFunc{}

func RegisterTemplate(name string, subjectFn templateFunc, channelBodies map[abstract.ChannelType]templateFunc) {
	if subjectFn != nil {
		subjects[name] = subjectFn
	}
	if bodies[name] == nil {
		bodies[name] = make(map[abstract.ChannelType]templateFunc)
	}
	for ct, fn := range channelBodies {
		bodies[name][ct] = fn
	}
}

func RenderSubject(name string, data map[string]any) string {
	fn, ok := subjects[name]
	if !ok {
		return name
	}
	return fn(data)
}

func RenderBody(ch abstract.ChannelType, name string, data map[string]any) string {
	chMap, ok := bodies[name]
	if !ok {
		return ""
	}
	fn, ok := chMap[ch]
	if !ok {
		return ""
	}
	return fn(data)
}

func init() {
	RegisterTemplate("password_reset",
		func(data map[string]any) string {
			return "Password Reset"
		},
		map[abstract.ChannelType]templateFunc{
			abstract.ChannelEmail: func(data map[string]any) string {
				token, _ := data["token"].(string)
				appURL, _ := data["app_url"].(string)
				resetURL := appURL + "/auth?token=" + url.QueryEscape(token)
				return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: sans-serif; max-width: 480px; margin: 0 auto; padding: 2rem;">
<h2>Password Reset</h2>
<p>Click the link below to reset your password. This link expires in 5 minutes.</p>
<p><a href="%s" style="display: inline-block; padding: 12px 24px; background: #0066cc; color: white; text-decoration: none; border-radius: 6px;">Reset Password</a></p>
<p>If you did not request this, ignore this email.</p>
</body>
</html>`, resetURL)
			},
			abstract.ChannelInApp: func(data map[string]any) string {
				return "A password reset was requested for your account."
			},
		},
	)
}
