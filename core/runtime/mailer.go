// @note #arch-20260821-009 issue resolved status=open priority=P2 tags=#arch,#maintainability : Hardcoded HTML template in Go source
//
// The password reset email HTML template is a fmt.Sprintf string literal
// (mailer.go:89-98). This makes it difficult to maintain, test, and internationalize.
//
// The existing TODO at line 83 acknowledges this issue.
//
// Resolution: Move templates to a templates/ directory with .html files,
// use Go's html/template package for rendering, support i18n for different
// locales, and make the template configurable per deployment.
package runtime

import (
	_ "embed"
	"html/template"
	"net/url"
	"strings"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/wneessen/go-mail"
)

//go:embed templates/password_reset.html
var passwordResetTmplRaw string

var passwordResetTmpl = template.Must(template.New("password_reset").Parse(passwordResetTmplRaw))

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
	client               *mail.Client
	from                 string
	passwordResetTmpl    *template.Template
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
		// S-21: Opportunistic TLS let a downgrade attacker capture reset
		// links and credentials in plaintext. Required by default; an
		// explicit opt-out is a deliberate code change, not an accident.
		mail.WithTLSPolicy(mail.TLSMandatory),
		mail.WithSMTPAuth(parseSMTPAuth(cfg.SMTPAuthType)),
		mail.WithUsername(cfg.SMTPUsername),
		mail.WithPassword(cfg.SMTPPassword),
	)
	if err != nil {
		return nil, common.SystemErrorFrom(err).WithOperation("NewMailer").WithMessage("create mail client")
	}
	from := cfg.FromAddress
	if cfg.FromName != "" {
		from = cfg.FromName + " <" + from + ">"
	}
	return &Mailer{client: client, from: from, passwordResetTmpl: passwordResetTmpl}, nil
}

// SetPasswordResetTemplate overrides the default password-reset email template.
// Pass nil to restore the embedded default.
func (m *Mailer) SetPasswordResetTemplate(t *template.Template) {
	if t != nil {
		m.passwordResetTmpl = t
	} else {
		m.passwordResetTmpl = passwordResetTmpl
	}
}

func (m *Mailer) Send(to, subject, body string) error {
	msg := mail.NewMsg()
	if err := msg.From(m.from); err != nil {
		return common.SystemErrorFrom(err).WithOperation("Mailer.Send").WithMessage("set from")
	}
	if err := msg.To(to); err != nil {
		return common.SystemErrorFrom(err).WithOperation("Mailer.Send").WithMessage("set to")
	}
	msg.Subject(subject)
	msg.SetBodyString(mail.TypeTextHTML, body)
	return m.client.DialAndSend(msg)
}

// @note #review-20260821-005 todo resolved status=open priority=P1 tags=#review,#maintainability : Extract email templates to separate files
// The HTML template in SendPasswordReset is hardcoded as a string literal.
// This makes it difficult to maintain, test, and internationalize.
//
// Consider:
// 1. Moving templates to a templates/ directory with .html files
// 2. Using Go's html/template package for rendering
// 3. Supporting i18n for different locales
// 4. Making the template configurable per deployment
func (m *Mailer) SendPasswordReset(email, token, appURL string) error {
	resetURL, err := url.JoinPath(appURL, "auth")
	if err != nil {
		return common.SystemErrorFrom(err).WithOperation("SendPasswordReset").WithMessage("build reset URL")
	}
	fullURL := resetURL + "?token=" + url.QueryEscape(token)

	var buf strings.Builder
	if err := m.passwordResetTmpl.Execute(&buf, map[string]string{"URL": fullURL}); err != nil {
		return common.SystemErrorFrom(err).WithOperation("SendPasswordReset").WithMessage("render template")
	}
	return m.Send(email, "Password Reset", buf.String())
}
