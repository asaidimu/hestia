package abstract

import (
	"context"
	"time"
)

type ChannelType string

const (
	ChannelEmail ChannelType = "email"
	ChannelInApp ChannelType = "in_app"
	ChannelSMS   ChannelType = "sms"
)

type Recipient struct {
	UserID string
	Email  string
	Phone  string
}

// NotificationAction is a suggested action attached to a notification. A
// client renders it as a button: Message (with Arguments) triggers a
// registered dispatcher message under the recipient's identity, URL is a plain
// link destination. At least one of Message or URL should be set.
type NotificationAction struct {
	// Label is the human-readable button text, e.g. "Apply update".
	Label string
	// Message is a registered dispatcher message name, e.g.
	// "system:updates:update:apply". Invocation is authorized by the policy
	// engine like any other dispatch.
	Message string
	// Arguments is the message input payload (arguments + body) to dispatch.
	Arguments map[string]any
	// URL is a link destination for actions that open a page instead of
	// dispatching a message.
	URL string
}

type Notification struct {
	ID        string
	TenantID  string
	Recipient Recipient
	Template  string
	Data      map[string]any
	Actions   []NotificationAction
	Channels  []ChannelType
	CreatedAt time.Time
}

type Channel interface {
	Type() ChannelType
	Send(ctx context.Context, n Notification) error
}

// TemplateResolver renders notification subject/body from a named template.
// Implementations load template definitions from storage and resolve them
// against the provided data using text/template.
type TemplateResolver interface {
	Render(ctx context.Context, channel ChannelType, name string, data map[string]any) (subject, body string, err error)
}

type Notifier interface {
	Send(ctx context.Context, n Notification) error
	RegisterChannel(ch Channel)
}
