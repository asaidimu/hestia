package notification

import (
	"context"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"

	"github.com/asaidimu/hestia/core/abstract"
)

const notificationsCollectionName = "_notifications_"

type inAppChannel struct {
	persist  base.Persistence
	resolver abstract.TemplateResolver
}

func NewInAppChannel(persist base.Persistence, resolver abstract.TemplateResolver) abstract.Channel {
	return &inAppChannel{persist: persist, resolver: resolver}
}

func (c *inAppChannel) Type() abstract.ChannelType { return abstract.ChannelInApp }

func (c *inAppChannel) Send(ctx context.Context, n abstract.Notification) error {
	col, err := c.persist.Collection(ctx, notificationsCollectionName)
	if err != nil {
		return err
	}

	subject, body, err := c.resolver.Render(ctx, abstract.ChannelInApp, n.Template, n.Data)
	if err != nil {
		return err
	}
	if body == "" {
		body = subject
	}

	doc := data.MustNewDocument(map[string]any{
		"user_id":    n.Recipient.UserID,
		"type":       n.Template,
		"subject":    subject,
		"body":       body,
		"data":       n.Data,
		"read":       false,
		"created_at": time.Now().UnixMilli(),
	})
	if len(n.Actions) > 0 {
		actions := make([]map[string]any, 0, len(n.Actions))
		for _, a := range n.Actions {
			m := map[string]any{"label": a.Label}
			if a.Message != "" {
				m["message"] = a.Message
				if len(a.Arguments) > 0 {
					m["arguments"] = a.Arguments
				}
			}
			if a.URL != "" {
				m["url"] = a.URL
			}
			actions = append(actions, m)
		}
		doc.Set("actions", actions)
	}
	if n.TenantID != "" {
		doc.Set("tenant_id", n.TenantID)
	}

	_, err = col.CreateOne(ctx, doc)
	return err
}
