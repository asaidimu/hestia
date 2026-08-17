package model

import (
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

// AuditLogEntryView is the declared shape of an audit log entry. Query and
// stream responses carry the raw persisted entries.
type AuditLogEntryView struct {
	EventID      string `anansi:"event_id"`
	OccurredAt   string `anansi:"occurred_at"`
	RecordedAt   string `anansi:"recorded_at"`
	TraceID      string `anansi:"trace_id"`
	RequestID    string `anansi:"request_id"`
	ActorID      string `anansi:"actor_id"`
	ActorType    string `anansi:"actor_type"`
	OnBehalfOfID string `anansi:"on_behalf_of_id"`
	AuthMethod   string `anansi:"auth_method"`
	SessionID    string `anansi:"session_id"`
	Operation    string `anansi:"operation"`
	ResourceType string `anansi:"resource_type"`
	ResourceID   string `anansi:"resource_id"`
	EventName    string `anansi:"event_name"`
	Status       string `anansi:"status"`
	Severity     string `anansi:"severity"`
	ErrorCode    string `anansi:"error_code"`
	ErrorMessage string `anansi:"error_message"`
	LatencyMs    int64  `anansi:"latency_ms"`
	SourceIP     string `anansi:"source_ip"`
	UserAgent    string `anansi:"user_agent"`
	ServiceName  string `anansi:"service_name"`
	Region       string `anansi:"region"`
}

// AuditPaginationView is the declared shape of the pagination metadata.
type AuditPaginationView struct {
	Total  int64  `anansi:"total"`
	Cursor string `anansi:"cursor"`
	Limit  int64  `anansi:"limit"`
}

// AuditLogPageView is the declared shape of a log query page.
type AuditLogPageView struct {
	Documents  []AuditLogEntryView `anansi:"documents"`
	Pagination AuditPaginationView `anansi:"pagination"`
}

// LogQueryOutput declares the paginated audit log schema.
type LogQueryOutput struct {
	Page AuditLogPageView `anansi:"page"`
}

func LogQueryOutputSchema() *definition.Schema { return dispatch.SchemaFromType[LogQueryOutput]() }

// LogStreamOutput declares the single log entry stream schema.
type LogStreamOutput struct {
	Document AuditLogEntryView `anansi:"document"`
}

func LogStreamOutputSchema() *definition.Schema { return dispatch.SchemaFromType[LogStreamOutput]() }