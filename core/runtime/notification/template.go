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
