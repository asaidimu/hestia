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
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/persistence/collection"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	apikeysmodel "github.com/asaidimu/hestia/core/system/apikeys/model"
	"github.com/asaidimu/hestia/core/system/auth"
	authmodel "github.com/asaidimu/hestia/core/system/auth/model"
	tenantsmodel "github.com/asaidimu/hestia/core/system/tenants/model"
	"github.com/asaidimu/hestia/core/system/users"
	usersmodel "github.com/asaidimu/hestia/core/system/users/model"
	"github.com/asaidimu/hestia/core/internal/testutil"
	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/runtime/notification"
)

func newUserModelOn(t *testing.T, p base.Persistence) *usersmodel.SystemUsers {
	t.Helper()
	usersmodel.DangerouslyResetSystemUsersModel()
	m, err := usersmodel.InitSystemUsersModel(p, zap.NewNop())
	if err != nil {
		t.Fatalf("InitSystemUsersModel: %v", err)
	}
	return m
}

type testMessage struct {
	name  string
	ctx   context.Context
	input data.Documenter
}

func (m testMessage) ID() string                            { return "" }
func (m testMessage) Name() string                          { return m.name }
func (m testMessage) Context() context.Context               { return m.ctx }
func (m testMessage) Input() data.Documenter                { return m.input }
func (m testMessage) InputChannel() <-chan abstract.StreamItem   { return nil }
func (m testMessage) BlobInputChannel() <-chan abstract.Blob { return nil }
func (m testMessage) TenantID() string                      { return "" }
func (m testMessage) TraceID() string                       { return "" }
func (m testMessage) RequestID() string                     { return "" }
func (m testMessage) SourceIP() string                      { return "" }
func (m testMessage) UserAgent() string                     { return "" }
func (m testMessage) ResourceID() string                    { return "" }
func (m testMessage) SessionID() string                     { return "" }

func newTestAuthService(t *testing.T, p base.Persistence, opts ...func(*auth.AuthService)) *auth.AuthService {
	t.Helper()
	userModel := newUserModelOn(t, p)
	tenantsmodel.DangerouslyResetSystemTenantsModel()
	tenantModel, err := tenantsmodel.InitSystemTenantsModel(p, zap.NewNop())
	if err != nil {
		t.Fatalf("InitSystemTenantsModel: %v", err)
	}
	sessionSvc := auth.NewSessionService("test-secret")
	credProv := auth.NewCredentialsProvider(sessionSvc, "test-secret:reset")

	ctx := context.Background()
	tenant, err := tenantModel.CreateTenant(ctx, "Test Tenant", "", nil)
	if err != nil {
		t.Fatalf("tenantModel.CreateTenant failed: %v", err)
	}
	tenantID := tenant.ID

	_, err = userModel.Register(ctx, "test@example.com", "secret123", "Test User", tenantID, nil)
	if err != nil {
		t.Fatalf("userModel.Register failed: %v", err)
	}

	apikeysmodel.DangerouslyResetSystemAPIKeysModel()
	apiKeyModel, err := apikeysmodel.InitSystemAPIKeysModel(p, zap.NewNop())
	if err != nil {
		t.Fatalf("InitSystemAPIKeysModel: %v", err)
	}

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

	apiKeyAuth := auth.NewAPIKeyAuthenticator(apiKeyModel, liveUsers, "ephemeral-key", "admin-1", "admin@example.com", zap.NewNop(), func() bool { return false })

	svc := auth.NewAuthServiceForTest(userModel, apiKeyModel, credProv, apiKeyAuth, "admin-1", 7*24*time.Hour)

	for _, opt := range opts {
		opt(svc)
	}

	return svc
}

func TestCreateSessionHandler(t *testing.T) {
	p := testutil.NewPersistence(t)
	svc := newTestAuthService(t, p)

	ctx := context.Background()
	msg := testMessage{name: "create-session", ctx: ctx}

	result, err := svc.CreateSession(ctx, msg, &authmodel.LoginInput{
		Email:    "test@example.com",
		Password: "secret123",
	})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if result == nil || result.Document == nil {
		t.Fatal("CreateSession returned nil result or document")
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
	svc := newTestAuthService(t, p)

	ctx := context.Background()
	msg := testMessage{name: "password-reset", ctx: ctx}

	t.Run("existing email returns success with nil mailer", func(t *testing.T) {
		err := svc.PasswordReset(ctx, msg, &authmodel.PasswordResetInput{
			Email: "test@example.com",
		})
		if err != nil {
			t.Fatalf("PasswordReset failed: %v", err)
		}
	})

	t.Run("non-existent email returns success (no enumeration)", func(t *testing.T) {
		err := svc.PasswordReset(ctx, msg, &authmodel.PasswordResetInput{
			Email: "nonexistent@example.com",
		})
		if err != nil {
			t.Fatalf("PasswordReset failed: %v", err)
		}
	})
}

func TestPasswordConfirmHandler(t *testing.T) {
	p := testutil.NewPersistence(t)
	svc := newTestAuthService(t, p)

	ctx := context.Background()
	msg := testMessage{name: "password-confirm", ctx: ctx}

	t.Run("valid token resets password", func(t *testing.T) {
		user, err := svc.TestUsers().GetByEmail(ctx, "test@example.com")
		if err != nil {
			t.Fatalf("GetByEmail failed: %v", err)
		}

		token, err := svc.TestCredProv().IssueResetToken(user.ID)
		if err != nil {
			t.Fatalf("IssueResetToken failed: %v", err)
		}

		err = svc.PasswordConfirm(ctx, msg, &authmodel.PasswordConfirmInput{
			Token:    token,
			Password: "newpass456",
		})
		if err != nil {
			t.Fatalf("PasswordConfirm failed: %v", err)
		}
	})

	t.Run("login with new password succeeds", func(t *testing.T) {
		loginMsg := testMessage{name: "create-session", ctx: ctx}
		result, err := svc.CreateSession(ctx, loginMsg, &authmodel.LoginInput{
			Email:    "test@example.com",
			Password: "newpass456",
		})
		if err != nil {
			t.Fatalf("login with new password failed: %v", err)
		}
		if result == nil || result.SessionToken == "" {
			t.Fatal("expected session token after password change")
		}
	})

	t.Run("login with old password fails", func(t *testing.T) {
		loginMsg := testMessage{name: "create-session", ctx: ctx}
		_, err := svc.CreateSession(ctx, loginMsg, &authmodel.LoginInput{
			Email:    "test@example.com",
			Password: "secret123",
		})
		if err == nil {
			t.Error("expected error when logging in with old password")
		}
	})

	t.Run("expired token returns error", func(t *testing.T) {
		user, err := svc.TestUsers().GetByEmail(ctx, "test@example.com")
		if err != nil {
			t.Fatalf("GetByEmail failed: %v", err)
		}

		now := time.Now().Add(-10 * time.Minute)
		exp := now.Add(5 * time.Minute).Unix()
		payload := fmt.Sprintf("%s:%d:%s", user.ID, exp, uuid.Must(uuid.NewV7()).String())
		mac := hmac.New(sha256.New, []byte("test-secret:reset"))
		mac.Write([]byte(payload))
		sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil)[:16])
		encoded := base64.RawURLEncoding.EncodeToString([]byte(payload))
		expiredToken := encoded + "." + sig

		err = svc.PasswordConfirm(ctx, msg, &authmodel.PasswordConfirmInput{
			Token:    expiredToken,
			Password: "anotherpass",
		})
		if err == nil {
			t.Error("expected error for expired token")
		}
	})

	t.Run("invalid signature returns error", func(t *testing.T) {
		err := svc.PasswordConfirm(ctx, msg, &authmodel.PasswordConfirmInput{
			Token:    "invalidsignature.token",
			Password: "anotherpass",
		})
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

	svc := newTestAuthService(t, p, func(s *auth.AuthService) {
		s.TestSetNotifier(notifier, "http://localhost:8070")
	})

	clearMailHog()

	ctx := context.Background()
	msg := testMessage{name: "password-reset", ctx: ctx}

	err = svc.PasswordReset(ctx, msg, &authmodel.PasswordResetInput{
		Email: "test@example.com",
	})
	if err != nil {
		t.Fatalf("PasswordReset: %v", err)
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
	apikeysmodel.DangerouslyResetSystemAPIKeysModel()
	apiKeyModel, err := apikeysmodel.InitSystemAPIKeysModel(p, zap.NewNop())
	if err != nil {
		t.Fatalf("InitSystemAPIKeysModel: %v", err)
	}

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

	a := auth.NewAPIKeyAuthenticator(apiKeyModel, liveUsers, "ephemeral-key", "admin-1", "admin@example.com", zap.NewNop(), func() bool { return false })
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
