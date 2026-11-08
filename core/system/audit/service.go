package audit

import (
	"context"

	"github.com/asaidimu/go-anansi/v8/core/document"
	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime/streamsink"
	"github.com/asaidimu/hestia/core/system/audit/model"
)

const auditCollectionName = "_audit_log_"

// AuditService handles audit log streaming.
type AuditService struct {
	persist persistence.Persistence
	logger  *zap.Logger
}

func NewAuditService(rt abstract.Container) (*AuditService, error) {
	persist := abstract.MustResolve[persistence.Persistence](rt)
	logger := abstract.MustResolve[*zap.Logger](rt)
	return &AuditService{persist: persist, logger: logger}, nil
}

// Stream streams audit log entries in real-time.
//
// @hestia.register(
//
//	name="system:audit:log:stream",
//	intent="stream",
//	rule="root",
//	description="Stream audit log entries in real-time",
//	input="model.LogStreamInput",
//	output="model.LogStreamOutput",
//
// )
func (s *AuditService) Stream(ctx context.Context, msg abstract.Message, input *model.LogStreamInput) (*abstract.Result, error) {
	sink := streamsink.New[*document.Document](64)

	col, err := s.persist.Collection(ctx, auditCollectionName)
	if err != nil {
		return nil, err
	}

	// Subscribe before returning so creates that land between the handler
	// returning and the readiness barrier firing are not lost. The barrier is
	// still observed so a stream that never goes live is torn down promptly.
	subID := col.Subscribe(ctx, persistence.SubscriptionOptions{
		Event: persistence.DocumentCreateSuccess,
		Callback: func(_ context.Context, event persistence.PersistenceEvent) error {
			outMap, ok := event.Output.(map[string]any)
			if !ok {
				return nil
			}
			dataRaw, ok := outMap["data"]
			if !ok {
				return nil
			}
			dataMap, ok := dataRaw.(map[string]any)
			if !ok || dataMap == nil {
				return nil
			}
			doc := document.NewRecordView(dataMap, context.Background())
			sink.TrySend(doc)
			return nil
		},
	})

	go func() {
		cleanup := func() {
			col.Unsubscribe(ctx, subID)
			sink.Close()
		}

		select {
		case <-msg.InputChannel():
		case <-ctx.Done():
			cleanup()
			// Client disconnected before the stream went live.
			return
		}

		<-ctx.Done()
		cleanup()
	}()

	return &abstract.Result{DocumentChannel: sink.Channel()}, nil
}
