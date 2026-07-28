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

type Notifier interface {
	Send(ctx context.Context, n Notification) error
	RegisterChannel(ch Channel)
}
