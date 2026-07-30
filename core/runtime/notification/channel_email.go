package notification

import (
	"context"
	"fmt"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
)

type emailChannel struct {
	mailer   *runtime.Mailer
	resolver abstract.TemplateResolver
}

func NewEmailChannel(mailer *runtime.Mailer, resolver abstract.TemplateResolver) abstract.Channel {
	return &emailChannel{mailer: mailer, resolver: resolver}
}

func (c *emailChannel) Type() abstract.ChannelType { return abstract.ChannelEmail }

func (c *emailChannel) Send(ctx context.Context, n abstract.Notification) error {
	if c.mailer == nil {
		return nil
	}

	subject, body, err := c.resolver.Render(ctx, abstract.ChannelEmail, n.Template, n.Data)
	if err != nil {
		return err
	}
	if body == "" {
		return fmt.Errorf("no template for %q / channel %q", n.Template, abstract.ChannelEmail)
	}

	return c.mailer.Send(n.Recipient.Email, subject, body)
}
