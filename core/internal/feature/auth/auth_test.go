package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/collection"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/internal/feature/apikeys"
	"github.com/asaidimu/hestia/core/internal/feature/auth"
	"github.com/asaidimu/hestia/core/internal/feature/tenants"
	"github.com/asaidimu/hestia/core/internal/feature/users"
	"github.com/asaidimu/hestia/core/internal/testutil"
	"github.com/asaidimu/hestia/core/runtime"
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
func (m testMessage) TenantID() string   { return "" }
func (m testMessage) TraceID() string    { return "" }
func (m testMessage) RequestID() string  { return "" }
func (m testMessage) SourceIP() string   { return "" }
func (m testMessage) UserAgent() string  { return "" }
func (m testMessage) ResourceID() string { return "" }
func (m testMessage) SessionID() string  { return "" }

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
	ctx = runtime.ContextWithTenantID(ctx, tenantID)

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

	_, err = userModel.Register(ctx, "test@example.com", "secret123", "Test User", tenantID)
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
