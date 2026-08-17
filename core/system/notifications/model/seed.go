package model

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"
)

var defaultNotificationTemplates = map[string]any{
	"notify:template:password_reset": map[string]any{
		"subject": "Password Reset",
		"bodies": map[string]string{
			"email": `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: sans-serif; max-width: 480px; margin: 0 auto; padding: 2rem;">
<h2>Password Reset</h2>
<p>Click the link below to reset your password. This link expires in 5 minutes.</p>
<p><a href="{{ .app_url }}/auth?token={{ .token }}" style="display: inline-block; padding: 12px 24px; background: #0066cc; color: white; text-decoration: none; border-radius: 6px;">Reset Password</a></p>
<p>If you did not request this, ignore this email.</p>
</body>
</html>`,
			"in_app": "A password reset was requested for your account.",
		},
	},
}

func SeedNotificationTemplates(ctx context.Context, persist base.Persistence) error {
	col, err := persist.Collection(ctx, "_settings_")
	if err != nil {
		return err
	}
	for key, val := range defaultNotificationTemplates {
		q := query.NewQueryBuilder().Where("key").Eq(key).Build()
		result, err := col.Read(ctx, &q)
		if err != nil {
			return err
		}
		if result.Count > 0 {
			continue
		}
		doc := data.MustNewDocument(map[string]any{
			"key":   key,
			"value": val,
		})
		if _, err := col.CreateOne(ctx, doc); err != nil {
			return fmt.Errorf("seed %q: %w", key, err)
		}
	}
	return nil
}
