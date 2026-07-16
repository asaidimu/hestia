package core

import "github.com/asaidimu/hestia/app/abstract"

type Message = abstract.Message
type Dispatcher = abstract.Dispatcher
type Registry = abstract.Registry
type MessageHandler = abstract.MessageHandler
type HandlerInfo = abstract.HandlerInfo
type IntentType = abstract.IntentType

const (
	IntentTypeCommand IntentType = abstract.IntentTypeCommand
	IntentTypeQuery   IntentType = abstract.IntentTypeQuery
)

type ResourceContextExtractor = abstract.ResourceContextExtractor
