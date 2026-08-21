// @note #arch-20260821-008 issue status=open priority=P1 tags=#arch,#dependency : Subpackage imports parent runtime package
//
// notification/channel_email.go imports its own parent runtime package (for *runtime.Mailer).
// This violates expected hierarchical dependency direction — subpackages should not import
// their parent.
//
// The Mailer type (or its interface) should live in a shared location like:
// - A new `core/config` package (alongside other config types)
// - The `abstract` package (as an interface)
// - A dedicated `core/mailer` package
//
// Resolution: Extract a Mailer interface in abstract or the new config package
// so notification does not need to import runtime.
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
