package core

import "context"

type ActorType string

const (
	ActorTypeUser      ActorType = "user"
	ActorTypeService   ActorType = "service"
	ActorTypeSystem    ActorType = "system"
	ActorTypeAnonymous ActorType = "anonymous"
)

type AuthMethod string

const (
	AuthMethodPassword       AuthMethod = "password"
	AuthMethodOAuth          AuthMethod = "oauth"
	AuthMethodAPIKey         AuthMethod = "api_key"
	AuthMethodMutualTLS      AuthMethod = "mtls"
	AuthMethodSSO            AuthMethod = "sso"
	AuthMethodServiceAccount AuthMethod = "service_account"
	AuthMethodNone           AuthMethod = "none"
)

type Operation string

const (
	OperationCreate  Operation = "create"
	OperationRead    Operation = "read"
	OperationUpdate  Operation = "update"
	OperationDelete  Operation = "delete"
	OperationLogin   Operation = "login"
	OperationLogout  Operation = "logout"
	OperationGrant   Operation = "grant"
	OperationRevoke  Operation = "revoke"
	OperationExecute Operation = "execute"
	OperationOther   Operation = "other"
)

type AuditStatus string

const (
	AuditStatusSuccess AuditStatus = "success"
	AuditStatusFailure AuditStatus = "failure"
	AuditStatusDenied  AuditStatus = "denied"
	AuditStatusError   AuditStatus = "error"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

type AuditEntry struct {
	EventID      string         `json:"event_id"`
	OccurredAt   string         `json:"occurred_at"`
	RecordedAt   string         `json:"recorded_at"`
	TraceID      string         `json:"trace_id,omitempty"`
	RequestID    string         `json:"request_id,omitempty"`
	ActorID      string         `json:"actor_id"`
	ActorType    ActorType      `json:"actor_type"`
	OnBehalfOfID string         `json:"on_behalf_of_id,omitempty"`
	AuthMethod   AuthMethod     `json:"auth_method,omitempty"`
	SessionID    string         `json:"session_id,omitempty"`
	Operation    Operation      `json:"operation"`
	ResourceType string         `json:"resource_type"`
	ResourceID   string         `json:"resource_id,omitempty"`
	EventName    string         `json:"event_name"`
	Status       AuditStatus    `json:"status"`
	Severity     Severity       `json:"severity,omitempty"`
	ErrorCode    string         `json:"error_code,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty"`
	LatencyMs    int64          `json:"latency_ms"`
	SourceIP     string         `json:"source_ip,omitempty"`
	UserAgent    string         `json:"user_agent,omitempty"`
	ServiceName  string         `json:"service_name"`
	Region       string         `json:"region,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type AuditPersister interface {
	Insert(ctx context.Context, entry AuditEntry) error
}
