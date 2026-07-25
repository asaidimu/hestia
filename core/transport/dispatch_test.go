package transport

import (
	"context"
	"os"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/registration"
)

func TestMain(m *testing.M) {
	_ = data.ConfigureDocumentFactory(data.DocumentFactoryConfig{}, zap.NewNop())
	os.Exit(m.Run())
}

func TestDispatch_Success(t *testing.T) {
	disp := &mockDispatcher{
		sendFn: func(msg abstract.Message) (*registration.Result, error) {
			if msg.Name() != "test:ping" {
				t.Errorf("name = %q, want test:ping", msg.Name())
			}
			if msg.ID() == "" {
				t.Error("expected non-empty ID")
			}
			return &registration.Result{}, nil
		},
	}

	result, err := Dispatch(disp, Input{
		Name:    "test:ping",
		Context: context.Background(),
	})
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestDispatch_WithCustomID(t *testing.T) {
	disp := &mockDispatcher{
		sendFn: func(msg abstract.Message) (*registration.Result, error) {
			if msg.ID() != "my-idempotency-key" {
				t.Errorf("id = %q, want my-idempotency-key", msg.ID())
			}
			return &registration.Result{}, nil
		},
	}

	Dispatch(disp, Input{
		Name:    "test:ping",
		Context: context.Background(),
		ID:      "my-idempotency-key",
	})
}

func TestDispatch_WithDocument(t *testing.T) {
	doc := data.MustNewDocument(map[string]any{"key": "val"})

	disp := &mockDispatcher{
		sendFn: func(msg abstract.Message) (*registration.Result, error) {
			input := msg.Input()
			if input == nil {
				t.Fatal("expected non-nil input")
			}
			v := input.GetOr("key", "")
			if v != "val" {
				t.Errorf("input.key = %v, want val", v)
			}
			return &registration.Result{}, nil
		},
	}

	Dispatch(disp, Input{
		Name:     "test:ping",
		Context:  context.Background(),
		Document: doc,
	})
}

func TestDispatch_StreamIntent(t *testing.T) {
	disp := &mockDispatcher{
		sendFn: func(msg abstract.Message) (*registration.Result, error) {
			if msg.InputChannel() == nil {
				t.Error("expected input channel for stream intent")
			}
			go func() {
				for range msg.InputChannel() {
				}
			}()
			return &registration.Result{}, nil
		},
	}

	Dispatch(disp, Input{
		Name:    "test:stream",
		Context: context.Background(),
		Intent:  registration.Stream,
	})
}

func TestDispatch_Error(t *testing.T) {
	disp := &mockDispatcher{
		sendFn: func(msg abstract.Message) (*registration.Result, error) {
			return nil, &dispatchError{"handler not found: test:missing"}
		},
	}

	_, err := Dispatch(disp, Input{
		Name:    "test:missing",
		Context: context.Background(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

type mockDispatcher struct {
	sendFn func(abstract.Message) (*registration.Result, error)
}

func (m *mockDispatcher) Send(msg abstract.Message) (*registration.Result, error) {
	if m.sendFn != nil {
		return m.sendFn(msg)
	}
	return &registration.Result{}, nil
}

type dispatchError struct{ s string }

func (e *dispatchError) Error() string { return e.s }
