package runtime

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/wneessen/go-mail"
)

type MailerConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPAuthType string
	FromAddress  string
	FromName     string
}

type Mailer struct {
	client *mail.Client
	from   string
}

func parseSMTPAuth(authType string) mail.SMTPAuthType {
	switch strings.ToLower(authType) {
	case "plain":
		return mail.SMTPAuthPlain
	case "login":
		return mail.SMTPAuthLogin
	case "cram-md5":
		return mail.SMTPAuthCramMD5
	case "none":
		return mail.SMTPAuthNoAuth
	default:
		return mail.SMTPAuthAutoDiscover
	}
}

func NewMailer(cfg MailerConfig) (*Mailer, error) {
	client, err := mail.NewClient(
		cfg.SMTPHost,
		mail.WithPort(cfg.SMTPPort),
		mail.WithTLSPolicy(mail.TLSOpportunistic),
		mail.WithSMTPAuth(parseSMTPAuth(cfg.SMTPAuthType)),
		mail.WithUsername(cfg.SMTPUsername),
		mail.WithPassword(cfg.SMTPPassword),
	)
	if err != nil {
		return nil, fmt.Errorf("create mail client: %w", err)
	}
	from := cfg.FromAddress
	if cfg.FromName != "" {
		from = cfg.FromName + " <" + from + ">"
	}
	return &Mailer{client: client, from: from}, nil
}

func (m *Mailer) Send(to, subject, body string) error {
	msg := mail.NewMsg()
	if err := msg.From(m.from); err != nil {
		return fmt.Errorf("set from: %w", err)
	}
	if err := msg.To(to); err != nil {
		return fmt.Errorf("set to: %w", err)
	}
	msg.Subject(subject)
	msg.SetBodyString(mail.TypeTextHTML, body)
	return m.client.DialAndSend(msg)
}

func (m *Mailer) SendPasswordReset(email, token, appURL string) error {
	// TODO extract ENDPOINT and email templates
	resetURL, err := url.JoinPath(appURL, "auth")
	if err != nil {
		return fmt.Errorf("build reset URL: %w", err)
	}
	fullURL := resetURL + "?token=" + url.QueryEscape(token)
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body style="font-family: sans-serif; max-width: 480px; margin: 0 auto; padding: 2rem;">
<h2>Password Reset</h2>
<p>Click the link below to reset your password. This link expires in 5 minutes.</p>
<p><a href="%s" style="display: inline-block; padding: 12px 24px; background: #0066cc; color: white; text-decoration: none; border-radius: 6px;">Reset Password</a></p>
<p>If you did not request this, ignore this email.</p>
</body>
</html>`, fullURL)
	return m.Send(email, "Password Reset", body)
}
