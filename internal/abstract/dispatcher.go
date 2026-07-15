package abstract

import "context"

type MessageHandler func(context.Context, Message) (*Result, error)

type Dispatcher interface {
	Send(msg Message) (*Result, error)
}

type IntentType string

const (
	IntentTypeCommand IntentType = "COMMAND"
	IntentTypeQuery   IntentType = "QUERY"
)

type HandlerInfo struct {
	Name        string     `json:"name"`
	IntentType  IntentType `json:"intent_type"`
	Description string     `json:"description"`
	Enabled     bool       `json:"enabled"`
}

type Registry interface {
	RegisterHandler(name string, handler MessageHandler, info HandlerInfo) error
	GetHandler(name string) (MessageHandler, error)
	DeleteHandler(name string) error
	ListHandlers() []HandlerInfo
	SetHandlerEnabled(name string, enabled bool) error
}

type ResourceContextExtractor interface {
	ResourceContext() any
}
