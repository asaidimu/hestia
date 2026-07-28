package runtime

import (
	"encoding/json"
	"io"
	"mime/quotedprintable"
	"net/http"
	"strings"
	"testing"
	"time"
)

type mailHogMsg struct {
	Content struct {
		Headers struct {
			To      []string `json:"To"`
			Subject []string `json:"Subject"`
		} `json:"Headers"`
		Body string `json:"Body"`
	} `json:"Content"`
}

type mailHogResp struct {
	Total int           `json:"total"`
	Items []mailHogMsg `json:"items"`
}

func mailHogLive() bool {
	resp, err := http.Get("http://localhost:8025/api/v2/messages")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return true
}

func clearMailHog() {
	req, _ := http.NewRequest("DELETE", "http://localhost:8025/api/v1/messages", nil)
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}

func pollMailHog(t *testing.T) mailHogMsg {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for MailHog message")
		default:
			resp, err := http.Get("http://localhost:8025/api/v2/messages")
			if err != nil {
				t.Fatalf("get MailHog: %v", err)
			}
			var mhr mailHogResp
			if err := json.NewDecoder(resp.Body).Decode(&mhr); err != nil {
				resp.Body.Close()
				t.Fatalf("decode MailHog: %v", err)
			}
			resp.Body.Close()
			if mhr.Total > 0 && len(mhr.Items) > 0 {
				return mhr.Items[0]
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
}

func decodeBody(qp string) string {
	r := quotedprintable.NewReader(strings.NewReader(qp))
	b, err := io.ReadAll(r)
	if err != nil {
		return qp
	}
	return string(b)
}

func TestMailer_SendPasswordReset(t *testing.T) {
	if !mailHogLive() {
		t.Skip("MailHog not running on localhost:1025")
	}
	clearMailHog()

	mailer, err := NewMailer(MailerConfig{
		SMTPHost:     "localhost",
		SMTPPort:     1025,
		SMTPAuthType: "none",
		FromAddress:  "noreply@test.local",
		FromName:     "Test App",
	})
	if err != nil {
		t.Fatalf("NewMailer: %v", err)
	}

	email := "admin@test.local"
	token := "abc123reset"
	appURL := "http://localhost:8070"
	if err := mailer.SendPasswordReset(email, token, appURL); err != nil {
		t.Fatalf("SendPasswordReset: %v", err)
	}

	msg := pollMailHog(t)

	to := strings.Trim(msg.Content.Headers.To[0], "<>")
	if to != email {
		t.Errorf("To = %q, want %q", to, email)
	}
	if len(msg.Content.Headers.Subject) == 0 || msg.Content.Headers.Subject[0] != "Password Reset" {
		t.Errorf("Subject = %v, want %q", msg.Content.Headers.Subject, "Password Reset")
	}

	body := decodeBody(msg.Content.Body)

	if !strings.Contains(body, "http://localhost:8070/auth?token=abc123reset") {
		t.Errorf("body missing reset URL:\n%s", body)
	}
	if !strings.Contains(body, "Reset Password") {
		t.Errorf("body missing button text:\n%s", body)
	}
	if !strings.Contains(body, "expires in 5 minutes") {
		t.Errorf("body missing expiry text:\n%s", body)
	}
}

func TestMailer_Send(t *testing.T) {
	if !mailHogLive() {
		t.Skip("MailHog not running on localhost:1025")
	}
	clearMailHog()

	mailer, err := NewMailer(MailerConfig{
		SMTPHost:     "localhost",
		SMTPPort:     1025,
		SMTPAuthType: "none",
		FromAddress:  "noreply@test.local",
		FromName:     "Test App",
	})
	if err != nil {
		t.Fatalf("NewMailer: %v", err)
	}

	if err := mailer.Send("user@test.local", "Hello", "<p>body</p>"); err != nil {
		t.Fatalf("Send: %v", err)
	}

	msg := pollMailHog(t)

	to := strings.Trim(msg.Content.Headers.To[0], "<>")
	if to != "user@test.local" {
		t.Errorf("To = %q, want %q", to, "user@test.local")
	}
	if len(msg.Content.Headers.Subject) == 0 || msg.Content.Headers.Subject[0] != "Hello" {
		t.Errorf("Subject = %v, want %q", msg.Content.Headers.Subject, "Hello")
	}

	body := decodeBody(msg.Content.Body)
	if !strings.Contains(body, "<p>body</p>") {
		t.Errorf("body missing expected content:\n%s", body)
	}
}
