package notification

import (
	"context"
	"fmt"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
)

type emailChannel struct {
	mailer *runtime.Mailer
}

func NewEmailChannel(mailer *runtime.Mailer) abstract.Channel {
	return &emailChannel{mailer: mailer}
}

func (c *emailChannel) Type() abstract.ChannelType { return abstract.ChannelEmail }

func (c *emailChannel) Send(ctx context.Context, n abstract.Notification) error {
	if c.mailer == nil {
		return nil
	}

	subject := RenderSubject(n.Template, n.Data)
	body := RenderBody(abstract.ChannelEmail, n.Template, n.Data)
	if body == "" {
		return fmt.Errorf("no template for %q / channel %q", n.Template, abstract.ChannelEmail)
	}

	return c.mailer.Send(n.Recipient.Email, subject, body)
}
