package core

import "context"

type AccessStatus string

const (
	AccessStatusOK      AccessStatus = "ok"
	AccessStatusDenied  AccessStatus = "denied"
	AccessStatusError   AccessStatus = "error"
)

type AccessLogEntry struct {
	Timestamp   string `json:"timestamp"`
	RequestID   string `json:"request_id"`
	UserID      string `json:"user_id,omitempty"`
	Credential  string `json:"credential,omitempty"`
	MessageName string `json:"message_name"`
	MessageID   string `json:"message_id"`
	Status      AccessStatus `json:"status"`
	Error       string `json:"error,omitempty"`
	LatencyMs   int64  `json:"latency_ms"`

	TransportMetadata any `json:"transport,omitempty"`
}

type AccessLogPersister interface {
	Insert(ctx context.Context, entry AccessLogEntry) error
}
