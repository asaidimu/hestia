package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/internal/feature/apikeys"
	"github.com/asaidimu/hestia/core/internal/feature/auth"
	"github.com/asaidimu/hestia/core/internal/feature/users"
	"github.com/asaidimu/hestia/core/internal/testutil"
)

type testMessage struct {
	name  string
	ctx   context.Context
	input *data.Document
}

func (m testMessage) ID() string                              { return "" }
func (m testMessage) Name() string                            { return m.name }
func (m testMessage) Context() context.Context                 { return m.ctx }
func (m testMessage) Input() *data.Document                    { return m.input }
func (m testMessage) InputChannel() <-chan *data.Document      { return nil }
func (m testMessage) BlobInputChannel() <-chan abstract.Blob   { return nil }

func TestRegisterHandler(t *testing.T) {
	p := testutil.NewPersistence(t)
	userModel := users.NewUserModel(p)
	handler := auth.NewRegisterHandler(userModel)

	ctx := context.Background()
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
	sessionSvc := auth.NewSessionService("test-secret")
	credProv := auth.NewCredentialsProvider(sessionSvc, "test-secret:reset")

	ctx := context.Background()
	_, err := userModel.Register(ctx, "test@example.com", "secret123", "Test User")
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

	token, st, err := svc.Create("user-1", 7*24*time.Hour)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if token == "" {
		t.Fatal("Create returned empty token")
	}
	if st.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", st.UserID, "user-1")
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
}

func TestSessionService_InvalidSignature(t *testing.T) {
	svc := auth.NewSessionService("test-secret")
	otherSvc := auth.NewSessionService("different-secret")

	token, _, err := svc.Create("user-1", 7*24*time.Hour)
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

	token, st, err := svc.Create("user-1", 7*24*time.Hour)
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

func TestNewAPIKeyAuthenticator(t *testing.T) {
	p := testutil.NewPersistence(t)
	userModel := users.NewUserModel(p)
	apiKeyModel := apikeys.NewAPIKeyModel(p)

	a := auth.NewAPIKeyAuthenticator(apiKeyModel, userModel, "ephemeral-key", "admin-1", "admin@example.com")
	if a == nil {
		t.Fatal("NewAPIKeyAuthenticator returned nil")
	}
}
