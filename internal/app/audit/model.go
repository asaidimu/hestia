package audit

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"

	"github.com/asaidimu/hestia/app/core"
)

const auditCollectionName = "_audit_log_"

type AuditModel struct {
	persistence base.Persistence
}

func NewAuditModel(persistence base.Persistence) *AuditModel {
	return &AuditModel{persistence: persistence}
}

func (m *AuditModel) collection(ctx context.Context) (base.Collection, error) {
	return m.persistence.Collection(ctx, auditCollectionName)
}

func (m *AuditModel) Insert(ctx context.Context, entry core.AuditEntry) error {
	col, err := m.collection(ctx)
	if err != nil {
		return fmt.Errorf("access audit_log collection: %w", err)
	}

	if entry.EventID == "" {
		entry.EventID = fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Int63())
	}
	if entry.OccurredAt == "" {
		entry.OccurredAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if entry.RecordedAt == "" {
		entry.RecordedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if entry.ServiceName == "" {
		entry.ServiceName = "hestia"
	}

	fields := map[string]any{
		"event_id":      entry.EventID,
		"occurred_at":   entry.OccurredAt,
		"recorded_at":   entry.RecordedAt,
		"actor_id":      entry.ActorID,
		"actor_type":    string(entry.ActorType),
		"operation":     string(entry.Operation),
		"resource_type": entry.ResourceType,
		"event_name":    entry.EventName,
		"status":        string(entry.Status),
		"latency_ms":    entry.LatencyMs,
		"service_name":  entry.ServiceName,
	}
	setIfNonEmpty(fields, "trace_id", entry.TraceID)
	setIfNonEmpty(fields, "request_id", entry.RequestID)
	setIfNonEmpty(fields, "on_behalf_of_id", entry.OnBehalfOfID)
	setIfNonEmpty(fields, "auth_method", string(entry.AuthMethod))
	setIfNonEmpty(fields, "session_id", entry.SessionID)
	setIfNonEmpty(fields, "resource_id", entry.ResourceID)
	setIfNonEmpty(fields, "severity", string(entry.Severity))
	setIfNonEmpty(fields, "error_code", entry.ErrorCode)
	setIfNonEmpty(fields, "error_message", entry.ErrorMessage)
	setIfNonEmpty(fields, "source_ip", entry.SourceIP)
	setIfNonEmpty(fields, "user_agent", entry.UserAgent)
	setIfNonEmpty(fields, "region", entry.Region)
	if entry.Metadata != nil {
		fields["metadata"] = entry.Metadata
	}

	doc := data.MustNewDocument(fields, ctx)
	_, err = col.CreateOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("create audit entry: %w", err)
	}
	return nil
}

var _ core.AuditPersister = (*AuditModel)(nil)

func setIfNonEmpty(m map[string]any, key, val string) {
	if val != "" {
		m[key] = val
	}
}
