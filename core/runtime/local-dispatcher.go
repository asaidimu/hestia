// @note #arch-20260821-005 issue resolved status=open priority=P1 tags=#arch,#errors : Inconsistent error handling in dispatchers
// @assignee opencode
// Replaced fmt.Errorf in LocalDispatcher and BootstrapDispatcher with SystemError sentinels (ErrNotFound, ErrAccessDenied, ErrAlreadyExists, ErrValidation). All error paths now produce structured, chainable errors.
//
// LocalDispatcher.Send uses fmt.Errorf for handler-not-found and disabled errors,
// but the project convention is to use common.SystemError for consistency with
// the rest of the codebase (see core/runtime/errors.go).
//
// The same issue exists in bootstrap-dispatcher.go.
//
// Meanwhile, secure-dispatcher.go and rate-limit.go correctly use ErrAccessDenied,
// ErrAuthRequired, ErrRateLimited etc.
//
// Resolution: Replace all fmt.Errorf handler-dispatch errors in local-dispatcher.go
// and bootstrap-dispatcher.go with appropriate SystemError sentinels (ErrNotFound,
// ErrValidation/ErrAccessDenied with WithOperation).
package runtime

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
)

var _ abstract.Dispatcher = (*LocalDispatcher)(nil)
var _ abstract.Registry = (*LocalDispatcher)(nil)

// PanicError wraps a recovered panic value and its stack for upstream
// classification. Upstream handlers can detect it via errors.As or errors.Is.
type PanicError struct {
	Recovered any
	Stack     []byte
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("panic: %v", e.Recovered)
}

// Unwrap returns the recovered value if it was an error, allowing
// errors.Is / errors.As to reach the original cause.
func (e *PanicError) Unwrap() error {
	if err, ok := e.Recovered.(error); ok {
		return err
	}
	return nil
}

type handlerEntry struct {
	fn            abstract.MessageHandler
	description   string
	enabled       bool
	bootstrapSafe bool
}

type LocalDispatcher struct {
	mu       sync.RWMutex
	handlers map[string]handlerEntry
	logger   *zap.Logger
}

func NewLocalDispatcher() *LocalDispatcher {
	return &LocalDispatcher{
		handlers: make(map[string]handlerEntry),
	}
}

func NewLocalDispatcherWithLogger(logger *zap.Logger) *LocalDispatcher {
	return &LocalDispatcher{
		handlers: make(map[string]handlerEntry),
		logger:   logger,
	}
}

// @note #review-20260821-001 todo status=open priority=P1 tags=#review,#errors : Use SystemError for handler dispatch errors
// LocalDispatcher.Send uses fmt.Errorf for handler-not-found and disabled errors,
// but the review guide recommends using common.SystemError for consistency with
// the rest of the codebase (see core/runtime/errors.go).
//
// Consider using ErrNotFound and ErrValidation/ErrAccessDenied with WithOperation
// for better error classification and client-facing error codes.
//
// Send accepts the message for asynchronous execution: the handler runs on a
// dedicated goroutine and onComplete is invoked exactly once with its outcome.
// Handler panics are recovered here and delivered as *PanicError, so no link
// in the chain needs its own recovery wrapper.
func (d *LocalDispatcher) Send(ctx context.Context, msg abstract.Message, onComplete abstract.CompletionFunc) error {
	if msg.Context() == nil {
		return ErrValidation.WithOperation(msg.Name()).WithCause(fmt.Errorf("message has nil context"))
	}
	d.mu.RLock()
	entry, ok := d.handlers[msg.Name()]
	d.mu.RUnlock()
	if !ok {
		return ErrNotFound.WithOperation(msg.Name())
	}
	if !entry.enabled {
		return ErrAccessDenied.WithOperation(msg.Name()).WithCause(fmt.Errorf("handler disabled"))
	}

	go func() {
		if err := ctx.Err(); err != nil {
			completeSafely(d.logger, onComplete, ctx, nil, err)
			return
		}
		res, err := executeHandler(entry.fn, msg, d.logger)
		completeSafely(d.logger, onComplete, ctx, res, err)
	}()
	return nil
}

// executeHandler runs fn with panic containment. A recovered panic is
// returned as *PanicError carrying the stack trace.
func executeHandler(fn abstract.MessageHandler, msg abstract.Message, logger *zap.Logger) (res *abstract.Result, err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := make([]byte, 4096)
			n := runtime.Stack(stack, false)
			if logger != nil {
				logger.Error("panic recovered in dispatcher",
					zap.String("message", msg.Name()),
					zap.Any("panic", r),
					zap.ByteString("stack", stack[:n]),
				)
			}
			res = nil
			err = &PanicError{Recovered: r, Stack: stack[:n]}
		}
	}()
	return fn(msg.Context(), msg)
}

// completeSafely delivers an outcome, containing panics raised by completion
// callbacks so a misbehaving consumer cannot take down the process.
func completeSafely(logger *zap.Logger, onComplete abstract.CompletionFunc, ctx context.Context, res *abstract.Result, err error) {
	defer func() {
		if r := recover(); r != nil && logger != nil {
			logger.Error("panic in dispatch completion callback", zap.Any("panic", r))
		}
	}()
	abstract.Complete(onComplete, ctx, res, err)
}

// @note #review-20260821-002 todo status=open priority=P1 tags=#review,#errors : Use SystemError for duplicate handler registration
// The error returned when a handler is already registered should use
// ErrAlreadyExists (or a similar SystemError) instead of fmt.Errorf for
// consistent error handling across the codebase.
func (d *LocalDispatcher) RegisterHandler(name string, handler abstract.MessageHandler, info abstract.HandlerInfo) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, exists := d.handlers[name]; exists {
		return ErrAlreadyExists.WithOperation(name)
	}
	d.handlers[name] = handlerEntry{fn: handler, description: info.Description, enabled: info.Enabled, bootstrapSafe: info.BootstrapSafe}
	return nil
}

func (d *LocalDispatcher) GetHandler(name string) (abstract.MessageHandler, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	entry, ok := d.handlers[name]
	if !ok {
		return nil, ErrNotFound.WithOperation(name)
	}
	return entry.fn, nil
}

func (d *LocalDispatcher) ListHandlers() []abstract.HandlerInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()
	result := make([]abstract.HandlerInfo, 0, len(d.handlers))
	for name, entry := range d.handlers {
		result = append(result, abstract.HandlerInfo{
			Name:          name,
			Description:   entry.description,
			Enabled:       entry.enabled,
			BootstrapSafe: entry.bootstrapSafe,
		})
	}
	return result
}

func (d *LocalDispatcher) SetHandlerEnabled(name string, enabled bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	entry, ok := d.handlers[name]
	if !ok {
		return ErrNotFound.WithOperation(name)
	}
	entry.enabled = enabled
	d.handlers[name] = entry
	return nil
}

func (d *LocalDispatcher) IsHandlerBootstrapSafe(name string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	entry, ok := d.handlers[name]
	if !ok {
		return false
	}
	return entry.bootstrapSafe
}

// DeleteHandler removes a handler. It is idempotent — deleting a non-existent
// handler returns nil, which is intentional for cleanup paths (e.g. removing
// ns-scoped handlers when a namespace is deleted).
func (d *LocalDispatcher) DeleteHandler(name string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.handlers, name)
	return nil
}
