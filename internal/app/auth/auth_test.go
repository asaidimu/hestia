package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/app/abstract"
	"github.com/asaidimu/hestia/internal/app/apikeys"
	"github.com/asaidimu/hestia/internal/app/auth"
	"github.com/asaidimu/hestia/internal/app/users"
	"github.com/asaidimu/hestia/app/core"
	"github.com/asaidimu/hestia/app/core/identity"
	"github.com/asaidimu/hestia/internal/utility/persistest"
)

type testMessage struct {
	name  string
	ctx   context.Context
	input *data.Document
}

func (m testMessage) ID() string                           { return "" }
func (m testMessage) Name() string                         { return m.name }
func (m testMessage) Context() context.Context              { return m.ctx }
func (m testMessage) Input() *data.Document                 { return m.input }
func (m testMessage) InputChannel() <-chan *data.Document   { return nil }
func (m testMessage) BlobInputChannel() <-chan abstract.Blob { return nil }

var _ core.Message = testMessage{}

func TestRegisterHandler(t *testing.T) {
	p := persistest.NewPersistence(t)
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
	p := persistest.NewPersistence(t)
	userModel := users.NewUserModel(p)
	jwtSvc := auth.NewJWTService("test-secret", 15*time.Minute, 7*24*time.Hour, 5*time.Minute)
	blocklist := auth.NewTokenBlocklistService(p)
	credProv := auth.NewCredentialsProvider(jwtSvc, blocklist, userModel)

	ctx := context.Background()
	_, err := userModel.Register(ctx, "test@example.com", "secret123", "Test User")
	if err != nil {
		t.Fatalf("userModel.Register failed: %v", err)
	}

	handler := auth.NewCreateSessionHandler(userModel, credProv)
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
	raw, err := result.Document.Get("token")
	if err != nil {
		t.Fatalf("result missing 'token' field: %v", err)
	}
	tokenMap, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("token is %T, want map[string]any", raw)
	}
	access, _ := tokenMap["access"].(string)
	if access == "" {
		t.Error("token.access is empty")
	}
}

func TestValidateTokenHandler(t *testing.T) {
	jwtSvc := auth.NewJWTService("test-secret", 15*time.Minute, 7*24*time.Hour, 5*time.Minute)
	p := persistest.NewPersistence(t)
	blocklist := auth.NewTokenBlocklistService(p)
	userModel := users.NewUserModel(p)
	credProv := auth.NewCredentialsProvider(jwtSvc, blocklist, userModel)

	token, err := jwtSvc.GenerateAccessToken("user-1", "test@example.com", []string{"read:*"})
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	handler := auth.NewValidateTokenHandler(credProv)
	ctx := context.Background()
	input := data.MustNewDocument(map[string]any{
		"token": token,
	}, ctx)
	msg := testMessage{name: "validate-token", ctx: ctx, input: input}

	result, err := handler(ctx, msg)
	if err != nil {
		t.Fatalf("ValidateTokenHandler failed: %v", err)
	}
	if result == nil || result.Document == nil {
		t.Fatal("ValidateTokenHandler returned nil result or document")
	}
	userID, err := result.Document.GetString("user_id")
	if err != nil {
		t.Fatalf("result missing user_id: %v", err)
	}
	if userID != "user-1" {
		t.Errorf("user_id = %q, want %q", userID, "user-1")
	}
	email, _ := result.Document.GetString("email")
	if email != "test@example.com" {
		t.Errorf("email = %q, want %q", email, "test@example.com")
	}
	tokenType, _ := result.Document.GetString("token_type")
	if tokenType != "access" {
		t.Errorf("token_type = %q, want %q", tokenType, "access")
	}
}

func TestDeleteSessionHandler(t *testing.T) {
	p := persistest.NewPersistence(t)
	userModel := users.NewUserModel(p)
	jwtSvc := auth.NewJWTService("test-secret", 15*time.Minute, 7*24*time.Hour, 5*time.Minute)
	blocklist := auth.NewTokenBlocklistService(p)

	ctx := context.Background()
	userDoc, err := userModel.Register(ctx, "test@example.com", "secret123", "Test User")
	if err != nil {
		t.Fatalf("userModel.Register failed: %v", err)
	}
	userID := userDoc.ID()

	token, err := jwtSvc.GenerateAccessToken(userID, "test@example.com", nil)
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}

	claims, err := jwtSvc.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	claimsCtx := identity.ContextWithClaims(ctx, claims)

	credProv := auth.NewCredentialsProvider(jwtSvc, blocklist, userModel)

	handler := auth.NewDeleteSessionHandler(credProv)
	input := data.MustNewDocument(map[string]any{
		"payload": map[string]any{},
	}, ctx)
	msg := testMessage{name: "delete-session", ctx: claimsCtx, input: input}

	_, err = handler(claimsCtx, msg)
	if err != nil {
		t.Fatalf("DeleteSessionHandler failed: %v", err)
	}

	blocklisted, err := blocklist.IsBlocklisted(ctx, claims.TokenID)
	if err != nil {
		t.Fatalf("IsBlocklisted failed: %v", err)
	}
	if !blocklisted {
		t.Error("expected token to be blocklisted after delete")
	}
}

func TestNewJWTService(t *testing.T) {
	svc := auth.NewJWTService("test-secret", 15*time.Minute, 7*24*time.Hour, 5*time.Minute)
	if svc == nil {
		t.Fatal("NewJWTService returned nil")
	}

	token, err := svc.GenerateAccessToken("user-1", "test@example.com", []string{"read:*"})
	if err != nil {
		t.Fatalf("GenerateAccessToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("GenerateAccessToken returned empty token")
	}

	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	if claims.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-1")
	}
	if claims.TokenType != "access" {
		t.Errorf("TokenType = %q, want %q", claims.TokenType, "access")
	}
}

func TestNewAPIKeyAuthenticator(t *testing.T) {
	p := persistest.NewPersistence(t)
	userModel := users.NewUserModel(p)
	apiKeyModel := apikeys.NewAPIKeyModel(p)

	a := auth.NewAPIKeyAuthenticator(apiKeyModel, userModel, "ephemeral-key", "admin-1", "admin@example.com")
	if a == nil {
		t.Fatal("NewAPIKeyAuthenticator returned nil")
	}
}
