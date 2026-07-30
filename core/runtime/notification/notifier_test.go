package notification

import (
	"context"
	"testing"

	"github.com/asaidimu/hestia/core/abstract"
)

type testChannel struct {
	typ  abstract.ChannelType
	sent []abstract.Notification
}

func (c *testChannel) Type() abstract.ChannelType { return c.typ }
func (c *testChannel) Send(_ context.Context, n abstract.Notification) error {
	c.sent = append(c.sent, n)
	return nil
}

type noopResolver struct{}

func (r *noopResolver) Render(_ context.Context, _ abstract.ChannelType, name string, _ map[string]any) (string, string, error) {
	return name, "", nil
}

func TestNotifier_SendsToRegisteredChannel(t *testing.T) {
	n := New(&noopResolver{})
	ch := &testChannel{typ: abstract.ChannelEmail}
	n.RegisterChannel(ch)

	err := n.Send(context.Background(), abstract.Notification{
		Recipient: abstract.Recipient{Email: "test@example.com"},
		Template:  "welcome",
		Channels:  []abstract.ChannelType{abstract.ChannelEmail},
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(ch.sent) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(ch.sent))
	}
	if ch.sent[0].Template != "welcome" {
		t.Errorf("template = %q, want %q", ch.sent[0].Template, "welcome")
	}
}

func TestNotifier_SkipsUnregisteredChannel(t *testing.T) {
	n := New(&noopResolver{})
	err := n.Send(context.Background(), abstract.Notification{
		Recipient: abstract.Recipient{Email: "test@example.com"},
		Template:  "welcome",
		Channels:  []abstract.ChannelType{abstract.ChannelSMS},
	})
	if err == nil {
		t.Error("expected error for unregistered channel")
	}
}

func TestNotifier_NoChannelsSkips(t *testing.T) {
	n := New(&noopResolver{})
	err := n.Send(context.Background(), abstract.Notification{
		Recipient: abstract.Recipient{Email: "test@example.com"},
		Template:  "welcome",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
}
