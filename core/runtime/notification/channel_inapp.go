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
	persist base.Persistence
}

func NewInAppChannel(persist base.Persistence) abstract.Channel {
	return &inAppChannel{persist: persist}
}

func (c *inAppChannel) Type() abstract.ChannelType { return abstract.ChannelInApp }

func (c *inAppChannel) Send(ctx context.Context, n abstract.Notification) error {
	col, err := c.persist.Collection(ctx, notificationsCollectionName)
	if err != nil {
		return err
	}

	subject := RenderSubject(n.Template, n.Data)
	body := RenderBody(abstract.ChannelInApp, n.Template, n.Data)
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
	if n.TenantID != "" {
		doc.Set("tenant_id", n.TenantID)
	}

	_, err = col.CreateOne(ctx, doc)
	return err
}
