package auth_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/quotedprintable"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/collection"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/internal/feature/apikeys"
	"github.com/asaidimu/hestia/core/internal/feature/auth"
	"github.com/asaidimu/hestia/core/internal/feature/tenants"
	"github.com/asaidimu/hestia/core/internal/feature/users"
	"github.com/asaidimu/hestia/core/internal/testutil"
	"github.com/asaidimu/hestia/core/runtime"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	"github.com/asaidimu/hestia/core/runtime/notification"
)

type testMessage struct {
	name  string
	ctx   context.Context
	input *data.Document
}

func (m testMessage) ID() string                             { return "" }
func (m testMessage) Name() string                           { return m.name }
func (m testMessage) Context() context.Context               { return m.ctx }
func (m testMessage) Input() *data.Document                  { return m.input }
func (m testMessage) InputChannel() <-chan *data.Document    { return nil }
func (m testMessage) BlobInputChannel() <-chan abstract.Blob { return nil }
func (m testMessage) TenantID() string                       { return "" }
func (m testMessage) TraceID() string                        { return "" }
func (m testMessage) RequestID() string                      { return "" }
func (m testMessage) SourceIP() string                       { return "" }
func (m testMessage) UserAgent() string                      { return "" }
func (m testMessage) ResourceID() string                     { return "" }
func (m testMessage) SessionID() string                      { return "" }

func TestRegisterHandler(t *testing.T) {
	p := testutil.NewPersistence(t)
	userModel := users.NewUserModel(p)
	tenantModel := tenants.NewTenantModel(p)

	ctx := context.Background()
	tenant, err := tenantModel.Create(ctx, "Test Tenant", "", nil)
	if err != nil {
		t.Fatalf("tenantModel.Create failed: %v", err)
	}
	tenantID := tenant.ID()
	ctx = runtimecontext.ContextWithTenantID(ctx, tenantID)

	handler := auth.NewRegisterHandler(userModel)
	input := data.MustNewDocument(map[string]any{
		"payload": map[string]any{
			"email":    "test@example.com",
			"password": "secret123",
			"name":     "Test User",
		},
	}, ctx)
	msg := testMessage{name: "register", ctx: ctx, input: input}

	result, err := handler(ctx, msg)
	if err != nil {
		t.Fatalf("RegisterHandler failed: %v", err)
	}
	if result == nil || result.Document == nil {
		t.Fatal("RegisterHandler returned nil result or document")
	}
	email, err := result.Document.GetString("email")
	if err != nil {
		t.Fatalf("result document missing email field: %v", err)
	}
	if email != "test@example.com" {
		t.Errorf("email = %q, want %q", email, "test@example.com")
	}
}

func TestCreateSessionHandler(t *testing.T) {
	p := testutil.NewPersistence(t)
	userModel := users.NewUserModel(p)
	tenantModel := tenants.NewTenantModel(p)
	sessionSvc := auth.NewSessionService("test-secret")
	credProv := auth.NewCredentialsProvider(sessionSvc, "test-secret:reset")

	ctx := context.Background()
	tenant, err := tenantModel.Create(ctx, "Test Tenant", "", nil)
	if err != nil {
		t.Fatalf("tenantModel.Create failed: %v", err)
	}
	tenantID := tenant.ID()

	_, err = userModel.Register(ctx, "test@example.com", "secret123", "Test User", tenantID, nil)
	if err != nil {
		t.Fatalf("userModel.Register failed: %v", err)
	}

	handler := auth.NewCreateSessionHandler(userModel, credProv, 7*24*time.Hour)
	input := data.MustNewDocument(map[string]any{
		"payload": map[string]any{
			"email":    "test@example.com",
			"password": "secret123",
		},
	}, ctx)
	msg := testMessage{name: "create-session", ctx: ctx, input: input}

	result, err := handler(ctx, msg)
	if err != nil {
		t.Fatalf("CreateSessionHandler failed: %v", err)
	}
	if result == nil || result.Document == nil {
		t.Fatal("CreateSessionHandler returned nil result or document")
	}
	if result.SessionToken == "" {
		t.Error("SessionToken is empty")
	}
}

func TestSessionService(t *testing.T) {
	svc := auth.NewSessionService("test-secret")

	token, st, err := svc.Create("user-1", 7*24*time.Hour, 0)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if token == "" {
		t.Fatal("Create returned empty token")
	}
	if st.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", st.UserID, "user-1")
	}
	if st.TokenVersion != 0 {
		t.Errorf("TokenVersion = %d, want 0", st.TokenVersion)
	}

	validated, err := svc.Validate(token)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if validated.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", validated.UserID, "user-1")
	}
	if validated.SessionID == "" {
		t.Error("SessionID is empty")
	}
	if validated.ExpiresAt <= validated.IssuedAt {
		t.Error("ExpiresAt should be after IssuedAt")
	}
	if validated.TokenVersion != 0 {
		t.Errorf("TokenVersion after validate = %d, want 0", validated.TokenVersion)
	}
}

func TestSessionService_InvalidSignature(t *testing.T) {
	svc := auth.NewSessionService("test-secret")
	otherSvc := auth.NewSessionService("different-secret")

	token, _, err := svc.Create("user-1", 7*24*time.Hour, 0)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	_, err = otherSvc.Validate(token)
	if err == nil {
		t.Error("expected error for token signed with different secret")
	}
}

func TestSessionService_Refresh(t *testing.T) {
	svc := auth.NewSessionService("test-secret")

	token, st, err := svc.Create("user-1", 7*24*time.Hour, 0)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	time.Sleep(1 * time.Second)

	newToken, refreshed, err := svc.Refresh(st)
	if err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}
	if newToken == token {
		t.Error("Refresh should return a new token string")
	}

	validated, err := svc.Validate(newToken)
	if err != nil {
		t.Fatalf("Validate refreshed token failed: %v", err)
	}
	if validated.SessionID != refreshed.SessionID {
		t.Error("SessionID should remain the same after refresh")
	}
	if validated.IssuedAt != refreshed.IssuedAt {
		t.Error("validated IssuedAt should match refreshed IssuedAt")
	}
	if refreshed.IssuedAt <= st.IssuedAt {
		t.Error("IssuedAt should be updated after refresh")
	}
	if validated.ExpiresAt != st.ExpiresAt {
		t.Error("ExpiresAt should remain the same after refresh")
	}
	if validated.CreatedAt != st.CreatedAt {
		t.Error("CreatedAt should remain the same after refresh")
	}
}

func TestSessionService_TokenVersion(t *testing.T) {
	svc := auth.NewSessionService("test-secret")

	t.Run("create with version embeds and preserves version", func(t *testing.T) {
		token, st, err := svc.Create("user-2", 7*24*time.Hour, 42)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if st.TokenVersion != 42 {
			t.Errorf("TokenVersion = %d, want 42", st.TokenVersion)
		}

		validated, err := svc.Validate(token)
		if err != nil {
			t.Fatalf("Validate failed: %v", err)
		}
		if validated.TokenVersion != 42 {
			t.Errorf("validated TokenVersion = %d, want 42", validated.TokenVersion)
		}
	})

	t.Run("refresh preserves version", func(t *testing.T) {
		_, st, err := svc.Create("user-2", 7*24*time.Hour, 7)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		_, refreshed, err := svc.Refresh(st)
		if err != nil {
			t.Fatalf("Refresh failed: %v", err)
		}
		if refreshed.TokenVersion != 7 {
			t.Errorf("refreshed TokenVersion = %d, want 7", refreshed.TokenVersion)
		}
	})
}

func TestPasswordResetHandler(t *testing.T) {
	p := testutil.NewPersistence(t)
	userModel := users.NewUserModel(p)
	tenantModel := tenants.NewTenantModel(p)
	sessionSvc := auth.NewSessionService("test-secret")
	credProv := auth.NewCredentialsProvider(sessionSvc, "test-secret:reset")

	ctx := context.Background()
	tenant, err := tenantModel.Create(ctx, "Test Tenant", "", nil)
	if err != nil {
		t.Fatalf("tenantModel.Create failed: %v", err)
	}
	tenantID := tenant.ID()

	_, err = userModel.Register(ctx, "test@example.com", "secret123", "Test User", tenantID, nil)
	if err != nil {
		t.Fatalf("userModel.Register failed: %v", err)
	}

	t.Run("existing email returns success with nil mailer", func(t *testing.T) {
		handler := auth.NewPasswordResetHandler(userModel, credProv, nil, "")
		input := data.MustNewDocument(map[string]any{
			"payload": map[string]any{"email": "test@example.com"},
		}, ctx)
		msg := testMessage{name: "password-reset", ctx: ctx, input: input}

		result, err := handler(ctx, msg)
		if err != nil {
			t.Fatalf("PasswordResetHandler failed: %v", err)
		}
		if result == nil {
			t.Fatal("PasswordResetHandler returned nil result")
		}
	})

	t.Run("non-existent email returns success (no enumeration)", func(t *testing.T) {
		handler := auth.NewPasswordResetHandler(userModel, credProv, nil, "")
		input := data.MustNewDocument(map[string]any{
			"payload": map[string]any{"email": "nonexistent@example.com"},
		}, ctx)
		msg := testMessage{name: "password-reset", ctx: ctx, input: input}

		result, err := handler(ctx, msg)
		if err != nil {
			t.Fatalf("PasswordResetHandler failed: %v", err)
		}
		if result == nil {
			t.Fatal("PasswordResetHandler returned nil result")
		}
	})
}

func TestPasswordConfirmHandler(t *testing.T) {
	p := testutil.NewPersistence(t)
	userModel := users.NewUserModel(p)
	tenantModel := tenants.NewTenantModel(p)
	sessionSvc := auth.NewSessionService("test-secret")
	credProv := auth.NewCredentialsProvider(sessionSvc, "test-secret:reset")

	ctx := context.Background()
	tenant, err := tenantModel.Create(ctx, "Test Tenant", "", nil)
	if err != nil {
		t.Fatalf("tenantModel.Create failed: %v", err)
	}
	tenantID := tenant.ID()

	_, err = userModel.Register(ctx, "test@example.com", "secret123", "Test User", tenantID, nil)
	if err != nil {
		t.Fatalf("userModel.Register failed: %v", err)
	}

	t.Run("valid token resets password", func(t *testing.T) {
		user, err := userModel.GetByEmail(ctx, "test@example.com")
		if err != nil {
			t.Fatalf("GetByEmail failed: %v", err)
		}

		token, err := credProv.IssueResetToken(user.ID())
		if err != nil {
			t.Fatalf("IssueResetToken failed: %v", err)
		}

		handler := auth.NewPasswordConfirmHandler(userModel, credProv)
		input := data.MustNewDocument(map[string]any{
			"payload": map[string]any{
				"token":    token,
				"password": "newpass456",
			},
		}, ctx)
		msg := testMessage{name: "password-confirm", ctx: ctx, input: input}

		result, err := handler(ctx, msg)
		if err != nil {
			t.Fatalf("PasswordConfirmHandler failed: %v", err)
		}
		if result == nil {
			t.Fatal("PasswordConfirmHandler returned nil result")
		}
	})

	t.Run("login with new password succeeds", func(t *testing.T) {
		loginHandler := auth.NewCreateSessionHandler(userModel, credProv, 7*24*time.Hour)
		input := data.MustNewDocument(map[string]any{
			"payload": map[string]any{
				"email":    "test@example.com",
				"password": "newpass456",
			},
		}, ctx)
		msg := testMessage{name: "create-session", ctx: ctx, input: input}

		result, err := loginHandler(ctx, msg)
		if err != nil {
			t.Fatalf("login with new password failed: %v", err)
		}
		if result == nil || result.SessionToken == "" {
			t.Fatal("expected session token after password change")
		}
	})

	t.Run("login with old password fails", func(t *testing.T) {
		loginHandler := auth.NewCreateSessionHandler(userModel, credProv, 7*24*time.Hour)
		input := data.MustNewDocument(map[string]any{
			"payload": map[string]any{
				"email":    "test@example.com",
				"password": "secret123",
			},
		}, ctx)
		msg := testMessage{name: "create-session", ctx: ctx, input: input}

		_, err := loginHandler(ctx, msg)
		if err == nil {
			t.Error("expected error when logging in with old password")
		}
	})

	t.Run("expired token returns error", func(t *testing.T) {
		user, err := userModel.GetByEmail(ctx, "test@example.com")
		if err != nil {
			t.Fatalf("GetByEmail failed: %v", err)
		}

		now := time.Now().Add(-10 * time.Minute)
		exp := now.Add(5 * time.Minute).Unix()
		payload := fmt.Sprintf("%s:%d:%s", user.ID(), exp, uuid.Must(uuid.NewV7()).String())
		mac := hmac.New(sha256.New, []byte("test-secret:reset"))
		mac.Write([]byte(payload))
		sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil)[:16])
		encoded := base64.RawURLEncoding.EncodeToString([]byte(payload))
		expiredToken := encoded + "." + sig

		handler := auth.NewPasswordConfirmHandler(userModel, credProv)
		input := data.MustNewDocument(map[string]any{
			"payload": map[string]any{
				"token":    expiredToken,
				"password": "anotherpass",
			},
		}, ctx)
		msg := testMessage{name: "password-confirm", ctx: ctx, input: input}

		_, err = handler(ctx, msg)
		if err == nil {
			t.Error("expected error for expired token")
		}
	})

	t.Run("invalid signature returns error", func(t *testing.T) {
		handler := auth.NewPasswordConfirmHandler(userModel, credProv)
		input := data.MustNewDocument(map[string]any{
			"payload": map[string]any{
				"token":    "invalidsignature.token",
				"password": "anotherpass",
			},
		}, ctx)
		msg := testMessage{name: "password-confirm", ctx: ctx, input: input}

		_, err := handler(ctx, msg)
		if err == nil {
			t.Error("expected error for invalid token signature")
		}
	})
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
	Total int          `json:"total"`
	Items []mailHogMsg `json:"items"`
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

func TestPasswordResetHandler_WithMailer(t *testing.T) {
	if !mailHogLive() {
		t.Skip("MailHog not running on localhost:1025")
	}

	p := testutil.NewPersistence(t)
	userModel := users.NewUserModel(p)
	tenantModel := tenants.NewTenantModel(p)
	sessionSvc := auth.NewSessionService("test-secret")
	credProv := auth.NewCredentialsProvider(sessionSvc, "test-secret:reset")

	ctx := context.Background()
	tenant, err := tenantModel.Create(ctx, "Test Tenant", "", nil)
	if err != nil {
		t.Fatalf("tenantModel.Create: %v", err)
	}
	tenantID := tenant.ID()

	_, err = userModel.Register(ctx, "test@example.com", "secret123", "Test User", tenantID, nil)
	if err != nil {
		t.Fatalf("userModel.Register: %v", err)
	}

	resolver := &testNotifResolver{}
	notifier := notification.New(resolver)
	mailer, err := runtime.NewMailer(runtime.MailerConfig{
		SMTPHost:     "localhost",
		SMTPPort:     1025,
		SMTPAuthType: "none",
		FromAddress:  "noreply@test.local",
		FromName:     "Test App",
	})
	if err != nil {
		t.Fatalf("NewMailer: %v", err)
	}
	notifier.RegisterChannel(notification.NewEmailChannel(mailer, resolver))

	clearMailHog()

	handler := auth.NewPasswordResetHandler(userModel, credProv, notifier, "http://localhost:8070")
	input := data.MustNewDocument(map[string]any{
		"payload": map[string]any{"email": "test@example.com"},
	}, ctx)
	msg := testMessage{name: "password-reset", ctx: ctx, input: input}

	result, err := handler(ctx, msg)
	if err != nil {
		t.Fatalf("PasswordResetHandler: %v", err)
	}
	if result == nil {
		t.Fatal("nil result")
	}

	emailMsg := pollMailHog(t)

	to := strings.Trim(emailMsg.Content.Headers.To[0], "<>")
	if to != "test@example.com" {
		t.Errorf("To = %q, want %q", to, "test@example.com")
	}
	if len(emailMsg.Content.Headers.Subject) == 0 || emailMsg.Content.Headers.Subject[0] != "Password Reset" {
		t.Errorf("Subject = %v, want %q", emailMsg.Content.Headers.Subject, "Password Reset")
	}

	body := decodeBody(emailMsg.Content.Body)
	if !strings.Contains(body, "http://localhost:8070/auth?token=") {
		t.Errorf("body missing reset URL:\n%s", body)
	}
	if !strings.Contains(body, "expires in 5 minutes") {
		t.Errorf("body missing expiry text:\n%s", body)
	}
}

func TestNewAPIKeyAuthenticator(t *testing.T) {
	p := testutil.NewPersistence(t)
	apiKeyModel := apikeys.NewAPIKeyModel(p)

	userColl, err := p.Collection(context.Background(), "_user_")
	if err != nil {
		t.Fatalf("open _user_ collection: %v", err)
	}
	liveUsers, err := collection.NewLiveRepository(context.Background(), collection.LiveRepositoryOptions[*users.UserClaims]{
		Collection: userColl,
		Processor:  &users.UserClaimsDocProcessor{},
		QueryKey:   "_id_",
		AutoLoad:   false,
	})
	if err != nil {
		t.Fatalf("create live user claims: %v", err)
	}
	t.Cleanup(func() { liveUsers.Close() })

	a := auth.NewAPIKeyAuthenticator(apiKeyModel, liveUsers, "ephemeral-key", "admin-1", "admin@example.com", zap.NewNop())
	if a == nil {
		t.Fatal("NewAPIKeyAuthenticator returned nil")
	}
}

type testNotifResolver struct{}

func (r *testNotifResolver) Render(_ context.Context, ch abstract.ChannelType, name string, data map[string]any) (string, string, error) {
	if ch == abstract.ChannelEmail && name == "password_reset" {
		token, _ := data["token"].(string)
		appURL, _ := data["app_url"].(string)
		body := fmt.Sprintf(`<!DOCTYPE html><html><body><a href="%s/auth?token=%s">Reset</a><p>expires in 5 minutes</p></body></html>`, appURL, token)
		return "Password Reset", body, nil
	}
	return name, "", nil
}
