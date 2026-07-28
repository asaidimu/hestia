package notification

import (
	"context"
	"testing"

	"github.com/asaidimu/hestia/core/abstract"
)

type testChannel struct {
	typ   abstract.ChannelType
	sent  []abstract.Notification
}

func (c *testChannel) Type() abstract.ChannelType { return c.typ }
func (c *testChannel) Send(_ context.Context, n abstract.Notification) error {
	c.sent = append(c.sent, n)
	return nil
}

func TestNotifier_SendsToRegisteredChannel(t *testing.T) {
	n := New()
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
	n := New()
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
	n := New()
	err := n.Send(context.Background(), abstract.Notification{
		Recipient: abstract.Recipient{Email: "test@example.com"},
		Template:  "welcome",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
}

func TestTemplateRender(t *testing.T) {
	subject := RenderSubject("password_reset", map[string]any{})
	if subject != "Password Reset" {
		t.Errorf("subject = %q, want %q", subject, "Password Reset")
	}

	body := RenderBody(abstract.ChannelEmail, "password_reset", map[string]any{
		"token":   "abc",
		"app_url": "http://test.local",
	})
	if body == "" {
		t.Fatal("empty body for password_reset/email")
	}
	if !contains(body, "http://test.local/auth?token=abc") {
		t.Errorf("body missing reset URL:\n%s", body)
	}

	missing := RenderBody(abstract.ChannelSMS, "password_reset", nil)
	if missing != "" {
		t.Errorf("expected empty body for unregistered channel, got %q", missing)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
