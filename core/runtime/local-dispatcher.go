package runtime

import (
	"fmt"
	"sync"

	"github.com/asaidimu/hestia/core/registration"
)

var _ Dispatcher = (*LocalDispatcher)(nil)
var _ Registry = (*LocalDispatcher)(nil)

type handlerEntry struct {
	fn          MessageHandler
	description string
	enabled     bool
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

func (d *LocalDispatcher) Send(msg Message) (*registration.Result, error) {
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

func (d *LocalDispatcher) RegisterHandler(name string, handler MessageHandler, info HandlerInfo) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, exists := d.handlers[name]; exists {
		return fmt.Errorf("handler already registered: %s", name)
	}
	d.handlers[name] = handlerEntry{fn: handler, description: info.Description, enabled: info.Enabled}
	return nil
}

func (d *LocalDispatcher) GetHandler(name string) (MessageHandler, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	entry, ok := d.handlers[name]
	if !ok {
		return nil, fmt.Errorf("handler not found: %s", name)
	}
	return entry.fn, nil
}

func (d *LocalDispatcher) ListHandlers() []HandlerInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()
	result := make([]HandlerInfo, 0, len(d.handlers))
	for name, entry := range d.handlers {
		result = append(result, HandlerInfo{
			Name:        name,
			Description: entry.description,
			Enabled:     entry.enabled,
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

func (d *LocalDispatcher) DeleteHandler(name string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.handlers, name)
	return nil
}
