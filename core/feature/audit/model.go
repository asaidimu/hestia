package audit

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"

	"github.com/asaidimu/hestia/core/internal/util"
	auditdomain "github.com/asaidimu/hestia/core/runtime/audit"
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

func (m *AuditModel) Insert(ctx context.Context, entry auditdomain.AuditEntry) error {
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

	fields := util.StructToMap(entry)

	doc := data.MustNewDocument(fields, ctx)
	_, err = col.CreateOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("create audit entry: %w", err)
	}
	return nil
}

var _ auditdomain.AuditPersister = (*AuditModel)(nil)
