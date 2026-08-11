package http

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

func TestMain(m *testing.M) {
	_ = data.ConfigureDocumentFactory(data.DocumentFactoryConfig{}, zap.NewNop())
	os.Exit(m.Run())
}

type mockTransport struct {
	mu       sync.Mutex
	handlers map[string]Handler
}

func newMockTransport() *mockTransport {
	return &mockTransport{handlers: make(map[string]Handler)}
}

func (m *mockTransport) Handle(pattern string, handler Handler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[pattern] = handler
}

func (m *mockTransport) Start() error                     { return nil }
func (m *mockTransport) Shutdown(_ context.Context) error { return nil }

type mockDispatcher struct {
	sendFn func(abstract.Message) (*abstract.Result, error)
}

func (m *mockDispatcher) Send(msg abstract.Message) (*abstract.Result, error) {
	if m.sendFn != nil {
		return m.sendFn(msg)
	}
	return &abstract.Result{}, nil
}

func mustPool(t *testing.T, s *definition.Schema) *document.DocumentPool {
	t.Helper()
	p, err := document.NewDocumentPool(s)
	if err != nil {
		t.Fatalf("NewDocumentPool: %v", err)
	}
	return p
}

func TestBuildDoc_PathParams(t *testing.T) {
	type userGetInput struct {
		UserID string `input:"arguments.user_id"`
	}
	input := runtime.Input{
		Schema:    dispatch.SchemaFromTypeWithTag[userGetInput]("input", true),
		Arguments: []abstract.ArgDef{{Name: "user_id", Type: definition.FieldTypeString}},
	}

	req := Request{
		PathParams: map[string]string{"user_id": "abc-123"},
	}

	doc, err := buildDoc(context.Background(), req, input, mustPool(t, input.Schema))
	if err != nil {
		t.Fatalf("buildDoc: %v", err)
	}
	userID, _ := doc.GetString("arguments.user_id")
	if userID != "abc-123" {
		t.Fatalf("expected user_id=abc-123, got %q", userID)
	}
}

func TestBuildDoc_PayloadObject(t *testing.T) {
	type renameInput struct {
		Payload map[string]any `input:"payload"`
	}
	input := runtime.Input{
		Schema:  dispatch.SchemaFromTypeWithTag[renameInput]("input", true),
		Payload: definition.FieldTypeObject,
	}

	req := Request{
		Body: []byte(`{"name": "New Name"}`),
	}

	doc, err := buildDoc(context.Background(), req, input, mustPool(t, input.Schema))
	if err != nil {
		t.Fatalf("buildDoc: %v", err)
	}
	payload, ok := doc.GetOr("payload", nil).(map[string]any)
	if !ok {
		t.Fatal("expected payload as map[string]any")
	}
	if payload["name"] != "New Name" {
		t.Fatalf("expected name=New Name, got %v", payload["name"])
	}
}

func TestBuildDoc_PayloadBytes(t *testing.T) {
	type uploadInput struct {
		Payload []byte `input:"payload"`
	}
	input := runtime.Input{
		Schema:  dispatch.SchemaFromTypeWithTag[uploadInput]("input", true),
		Payload: definition.FieldTypeBytes,
	}

	req := Request{
		Body: []byte("raw data"),
	}

	doc, err := buildDoc(context.Background(), req, input, mustPool(t, input.Schema))
	if err != nil {
		t.Fatalf("buildDoc: %v", err)
	}
	payload := doc.GetOr("payload", nil)
	b, ok := payload.([]byte)
	if !ok {
		t.Fatalf("expected payload as []byte, got %T", payload)
	}
	if string(b) != "raw data" {
		t.Fatalf("expected 'raw data', got %s", string(b))
	}
}

func TestBuildDoc_Modifiers(t *testing.T) {
	type notifyInput struct {
		Email string `input:"modifiers.email"`
	}
	input := runtime.Input{
		Schema:    dispatch.SchemaFromTypeWithTag[notifyInput]("input", true),
		Modifiers: map[string]definition.FieldType{"email": definition.FieldTypeString},
	}

	req := Request{
		Query: map[string][]string{"email": {"a@b.com"}},
	}

	doc, err := buildDoc(context.Background(), req, input, mustPool(t, input.Schema))
	if err != nil {
		t.Fatalf("buildDoc: %v", err)
	}
	email, _ := doc.GetString("modifiers.email")
	if email != "a@b.com" {
		t.Fatalf("expected email=a@b.com, got %q", email)
	}
}

func TestBuildDoc_DecodeError(t *testing.T) {
	type loginInput struct {
		Payload map[string]any `input:"payload"`
	}
	input := runtime.Input{
		Schema:  dispatch.SchemaFromTypeWithTag[loginInput]("input", true),
		Payload: definition.FieldTypeObject,
	}

	req := Request{
		Body: []byte(`{"email": `),
	}

	_, err := buildDoc(context.Background(), req, input, mustPool(t, input.Schema))
	if err == nil {
		t.Fatal("expected decode error for malformed body")
	}
}

func TestBuildDoc_ArgsOnlyWhenDeclared(t *testing.T) {
	type createInput struct {
		Payload map[string]any `input:"payload"`
	}
	input := runtime.Input{
		Schema:    dispatch.SchemaFromTypeWithTag[createInput]("input", true),
		Arguments: []abstract.ArgDef{{Name: "user_id", Type: definition.FieldTypeString}},
	}

	req := Request{
		PathParams: map[string]string{"user_id": "abc-123"},
		Body:       []byte(`{"name": "New Name"}`),
	}

	doc, err := buildDoc(context.Background(), req, input, mustPool(t, input.Schema))
	if err != nil {
		t.Fatalf("buildDoc: %v", err)
	}
	if doc.HasKey("arguments") {
		t.Fatal("arguments should not be emitted when the schema does not declare it")
	}
	if !doc.HasKey("payload") {
		t.Fatal("expected payload field")
	}
}

func TestSerializeResponse_Create(t *testing.T) {
	doc := data.MustNewDocument(map[string]any{"email": "a@b.com"})
	result := &abstract.Result{Document: doc}
	output := &definition.Schema{
		BaseSchema: definition.BaseSchema{
			Fields: map[definition.FieldId]definition.Field{
				"document": {FieldProperties: definition.FieldProperties{Type: definition.FieldTypeObject}},
			},
		},
	}
	resp := serializeResponse(result, output, abstract.Create, "/api/users")
	if resp.Status != 201 {
		t.Fatalf("expected 201, got %d", resp.Status)
	}
}

func TestSerializeResponse_Delete(t *testing.T) {
	resp := serializeResponse(nil, nil, abstract.Delete, "")
	if resp.Status != 204 {
		t.Fatalf("expected 204, got %d", resp.Status)
	}
}

func TestSerializeResponse_Read(t *testing.T) {
	doc := data.MustNewDocument(map[string]any{"id": "abc"})
	result := &abstract.Result{Document: doc}
	output := &definition.Schema{
		BaseSchema: definition.BaseSchema{
			Fields: map[definition.FieldId]definition.Field{
				"document": {FieldProperties: definition.FieldProperties{Type: definition.FieldTypeObject}},
			},
		},
	}
	resp := serializeResponse(result, output, abstract.Read, "")
	if resp.Status != 200 {
		t.Fatalf("expected 200, got %d", resp.Status)
	}
}

func TestSerializeResponse_Blob(t *testing.T) {
	result := &abstract.Result{
		Blob: abstract.Blob{
			Data:        []byte("blob-data"),
			ContentType: "text/plain",
		},
	}
	resp := serializeResponse(result, nil, abstract.Read, "")
	if resp.Status != 200 {
		t.Fatalf("expected 200, got %d", resp.Status)
	}
	if string(resp.Body.([]byte)) != "blob-data" {
		t.Fatalf("expected blob-data, got %v", resp.Body)
	}
	if resp.Headers["Content-Type"][0] != "text/plain" {
		t.Fatalf("expected Content-Type text/plain, got %v", resp.Headers["Content-Type"])
	}
}

func TestRegisterDispatcher_CreatesRoute(t *testing.T) {
	mt := newMockTransport()
	reg := abstract.MessageRegistration{
		Name: "test:user:profile:get",
		Handler: func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
			return &abstract.Result{}, nil
		},
		Intent: abstract.Read,
		Input: runtime.Input{
			Arguments: []abstract.ArgDef{{Name: "user_id", Type: definition.FieldTypeString}},
		},
	}

	orch := &Interface{
		trans:        mt,
		disp:         &mockDispatcher{},
		regs:         []abstract.MessageRegistration{reg},
		bootstrapped: true,
	}
	orch.installDispatcherRegistrations()

	mt.mu.Lock()
	_, ok := mt.handlers["GET /test/user/profile/get/{user_id}"]
	mt.mu.Unlock()
	if !ok {
		t.Fatal("expected route GET /test/user/profile/get/{user_id} to be registered")
	}
}

func TestRegisterDispatcher_QueryRoute(t *testing.T) {
	mt := newMockTransport()
	reg := abstract.MessageRegistration{
		Name: "test:user:profile:query",
		Handler: func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
			return &abstract.Result{}, nil
		},
		Intent: abstract.Query,
	}

	orch := &Interface{
		trans:        mt,
		disp:         &mockDispatcher{},
		regs:         []abstract.MessageRegistration{reg},
		bootstrapped: true,
	}
	orch.installDispatcherRegistrations()

	mt.mu.Lock()
	_, ok := mt.handlers["POST /test/user/profile/query"]
	mt.mu.Unlock()
	if !ok {
		t.Fatal("expected route POST /test/user/profile/query to be registered")
	}
}

func TestDeriveRoute_WithArguments(t *testing.T) {
	path := DeriveRoute("system:blobs:blob:download", []abstract.ArgDef{{Name: "ns", Type: definition.FieldTypeString}, {Name: "key", Type: definition.FieldTypeString}})
	if path != "/system/blobs/blob/download/{ns}/{key}" {
		t.Fatalf("expected /system/blobs/blob/download/{ns}/{key}, got %s", path)
	}
}

func TestDeriveRoute_NoArguments(t *testing.T) {
	path := DeriveRoute("system:core:health:check", nil)
	if path != "/system/core/health/check" {
		t.Fatalf("expected /system/core/health/check, got %s", path)
	}
}
