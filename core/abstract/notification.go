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

type Notification struct {
	ID        string
	TenantID  string
	Recipient Recipient
	Template  string
	Data      map[string]any
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
