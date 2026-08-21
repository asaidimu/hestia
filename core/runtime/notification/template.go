// @note #arch-20260821-012 issue status=open priority=P2 tags=#arch,#error-handling : Notification template resolver silently swallows errors
//
// The Render method silently returns (name, "", nil) for multiple error conditions:
// - Collection open failure (line 40-42)
// - Query failure or empty result (line 45-48)
// - Get failure on value field (line 50-53)
// - Type assertion failure (line 54-57)
// - Missing subject in template (line 72-74)
//
// This makes debugging template issues impossible in production. The caller
// receives an empty body with no indication of what went wrong.
//
// Resolution: Log warnings for these conditions or return a sentinel error
// that callers can handle differently from "template not found."
package notification

import (
	"context"

	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime/templateutil"
)

const settingsCollection = "_settings_"

type templateDoc struct {
	Subject string            `json:"subject"`
	Bodies  map[string]string `json:"bodies"`
}

type settingsResolver struct {
	persist base.Persistence
}

func NewSettingsResolver(persist base.Persistence) abstract.TemplateResolver {
	return &settingsResolver{persist: persist}
}

// @note #review-20260821-011 issue status=open priority=P2 tags=#review,#error-handling : Silent error swallowing in template resolution
// The Render method silently returns (name, "", nil) for multiple error conditions:
// - Collection open failure
// - Query failure or empty result
// - Get failure on value field
// - Missing subject in template
//
// This makes debugging template issues difficult. Consider logging warnings
// for these conditions or returning a sentinel error that callers can handle
// differently from "template not found."
func (r *settingsResolver) Render(ctx context.Context, channel abstract.ChannelType, name string, data map[string]any) (subject, body string, err error) {
	col, err := r.persist.Collection(ctx, settingsCollection)
	if err != nil {
		return name, "", nil
	}

	q := query.NewQueryBuilder().Where("key").Eq("notify:template:" + name).Build()
	result, err := col.Read(ctx, &q)
	if err != nil || result.Count == 0 {
		return name, "", nil
	}

	raw, err := result.Data[0].Get("value")
	if err != nil {
		return name, "", nil
	}
	val, ok := raw.(map[string]any)
	if !ok {
		return name, "", nil
	}

	tpl := templateDoc{}
	if s, _ := val["subject"].(string); s != "" {
		tpl.Subject = s
	}
	if b, _ := val["bodies"].(map[string]any); len(b) > 0 {
		tpl.Bodies = make(map[string]string, len(b))
		for k, v := range b {
			if s, ok := v.(string); ok {
				tpl.Bodies[k] = s
			}
		}
	}

	if tpl.Subject == "" {
		return name, "", nil
	}

	resolved := templateutil.ResolveMap(map[string]any{"subject": tpl.Subject, "body": tpl.Bodies[string(channel)]}, data)
	subject, _ = resolved["subject"].(string)
	body, _ = resolved["body"].(string)
	return subject, body, nil
}
