package model

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/document"

	auditdomain "github.com/asaidimu/hestia/core/runtime/audit"
)

// Insert satisfies audit.AuditPersister by mapping an AuditEntry to a
// SystemAuditLog document and persisting it.
func (m *SystemAuditLogs) Insert(ctx context.Context, entry auditdomain.AuditEntry) error {
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

	doc := document.New(&SystemAuditLog{
		EventID:      entry.EventID,
		OccurredAt:   entry.OccurredAt,
		RecordedAt:   entry.RecordedAt,
		ActorID:      entry.ActorID,
		ActorType:    ActorType(entry.ActorType),
		OnBehalfOfID: ptrStr(entry.OnBehalfOfID),
		AuthMethod:   authMethodPtr(entry.AuthMethod),
		SessionID:    ptrStr(entry.SessionID),
		Operation:    Operation(entry.Operation),
		ResourceType: entry.ResourceType,
		ResourceID:   ptrStr(entry.ResourceID),
		EventName:    entry.EventName,
		Status:       Status(entry.Status),
		Severity:     severityPtr(entry.Severity),
		ErrorCode:    ptrStr(entry.ErrorCode),
		ErrorMessage: ptrStr(entry.ErrorMessage),
		LatencyMs:    &entry.LatencyMs,
		SourceIp:     ptrStr(entry.SourceIP),
		UserAgent:    ptrStr(entry.UserAgent),
		ServiceName:  entry.ServiceName,
		Region:       ptrStr(entry.Region),
		TraceID:      ptrStr(entry.TraceID),
		RequestID:    ptrStr(entry.RequestID),
		Metadata:     entry.Metadata,
	})

	if _, err := m.Create(ctx, doc); err != nil {
		return fmt.Errorf("create audit entry: %w", err)
	}
	return nil
}

func ptrStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func authMethodPtr(m auditdomain.AuthMethod) *AuthMethod {
	if m == "" {
		return nil
	}
	v := AuthMethod(m)
	return &v
}

func severityPtr(s auditdomain.Severity) *Severity {
	if s == "" {
		return nil
	}
	v := Severity(s)
	return &v
}
