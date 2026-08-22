// @note #arch-20260821-005 issue status=open priority=P1 tags=#arch,#errors : Inconsistent error handling in dispatchers
//
// LocalDispatcher.Send uses fmt.Errorf for handler-not-found and disabled errors,
// but the project convention is to use common.SystemError for consistency with
// the rest of the codebase (see core/runtime/errors.go).
//
// The same issue exists in bootstrap-dispatcher.go line 29.
//
// Meanwhile, secure-dispatcher.go and rate-limit.go correctly use ErrAccessDenied,
// ErrAuthRequired, ErrRateLimited etc.
//
// Resolution: Replace all fmt.Errorf handler-dispatch errors in local-dispatcher.go
// and bootstrap-dispatcher.go with appropriate SystemError sentinels (ErrNotFound,
// ErrValidation/ErrAccessDenied with WithOperation).
package runtime

import (
	"fmt"
	"sync"

	"github.com/asaidimu/hestia/core/abstract"
)

var _ abstract.Dispatcher = (*LocalDispatcher)(nil)
var _ abstract.Registry = (*LocalDispatcher)(nil)

type handlerEntry struct {
	fn            abstract.MessageHandler
	description   string
	enabled       bool
	bootstrapSafe bool
}

type LocalDispatcher struct {
	mu       sync.RWMutex
	handlers map[string]handlerEntry
}

func NewLocalDispatcher() *LocalDispatcher {
	return &LocalDispatcher{
		handlers: make(map[string]handlerEntry),
	}
}

// @note #review-20260821-001 todo status=open priority=P1 tags=#review,#errors : Use SystemError for handler dispatch errors
// LocalDispatcher.Send uses fmt.Errorf for handler-not-found and disabled errors,
// but the review guide recommends using common.SystemError for consistency with
// the rest of the codebase (see core/runtime/errors.go).
//
// Consider using ErrNotFound and ErrValidation/ErrAccessDenied with WithOperation
// for better error classification and client-facing error codes.
func (d *LocalDispatcher) Send(msg abstract.Message) (*abstract.Result, error) {
	if msg.Context() == nil {
		return nil, fmt.Errorf("message %s has nil context", msg.Name())
	}
	d.mu.RLock()
	entry, ok := d.handlers[msg.Name()]
	d.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("handler not found: %s", msg.Name())
	}
	if !entry.enabled {
		return nil, fmt.Errorf("handler %s is disabled", msg.Name())
	}
	return entry.fn(msg.Context(), msg)
}

// @note #review-20260821-002 todo status=open priority=P1 tags=#review,#errors : Use SystemError for duplicate handler registration
// The error returned when a handler is already registered should use
// ErrAlreadyExists (or a similar SystemError) instead of fmt.Errorf for
// consistent error handling across the codebase.
func (d *LocalDispatcher) RegisterHandler(name string, handler abstract.MessageHandler, info abstract.HandlerInfo) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, exists := d.handlers[name]; exists {
		return fmt.Errorf("handler already registered: %s", name)
	}
	d.handlers[name] = handlerEntry{fn: handler, description: info.Description, enabled: info.Enabled, bootstrapSafe: info.BootstrapSafe}
	return nil
}

func (d *LocalDispatcher) GetHandler(name string) (abstract.MessageHandler, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	entry, ok := d.handlers[name]
	if !ok {
		return nil, fmt.Errorf("handler not found: %s", name)
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
		return fmt.Errorf("handler not found: %s", name)
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
