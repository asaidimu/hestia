package audit

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"

	"github.com/asaidimu/hestia/internal/core"
)

const accessLogCollectionName = "_access_log_"

type AccessLogModel struct {
	persistence base.Persistence
}

func NewAccessLogModel(persistence base.Persistence) *AccessLogModel {
	return &AccessLogModel{persistence: persistence}
}

func (m *AccessLogModel) collection(ctx context.Context) (base.Collection, error) {
	return m.persistence.Collection(ctx, accessLogCollectionName)
}

func (m *AccessLogModel) Insert(ctx context.Context, entry core.AccessLogEntry) error {
	col, err := m.collection(ctx)
	if err != nil {
		return fmt.Errorf("access access_log collection: %w", err)
	}

	doc := data.MustNewDocument(map[string]any{
		"timestamp":    entry.Timestamp,
		"request_id":   entry.RequestID,
		"user_id":      entry.UserID,
		"credential":   entry.Credential,
		"message_name": entry.MessageName,
		"message_id":   entry.MessageID,
		"status":       string(entry.Status),
		"error":        entry.Error,
		"latency_ms":   entry.LatencyMs,
	})
	if entry.TransportMetadata != nil {
		if err := doc.Set("transport", entry.TransportMetadata); err != nil {
			return fmt.Errorf("set transport metadata: %w", err)
		}
	}

	_, err = col.CreateOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("create access log entry: %w", err)
	}
	return nil
}

var _ core.AccessLogPersister = (*AccessLogModel)(nil)


