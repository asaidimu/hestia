package audit

import (
	"context"
	"sync"

	"github.com/asaidimu/go-anansi/v8/core/document"
	persistence "github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime/dispatch"
	"github.com/asaidimu/hestia/core/system/audit/model"
	"github.com/asaidimu/hestia/core/system/policies"
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
//	rule="administrator",
//	description="Stream audit log entries in real-time",
//	input="model.LogStreamInput",
//	output="model.LogStreamOutput",
//
// )
func (s *AuditService) Stream(ctx context.Context, msg abstract.Message, input *model.LogStreamInput) (*abstract.Result, error) {
	docCh := make(chan *document.Document, 64)

	col, err := s.persist.Collection(ctx, auditCollectionName)
	if err != nil {
		return nil, err
	}

	go func() {
		// close/unsubscribe must run at most once (a second close would panic
		// in an unrecovered goroutine). Mirrors the notifications Stream fix.
		var once sync.Once
		var subID string
		cleanup := func() {
			once.Do(func() {
				close(docCh)
				if subID != "" {
					col.Unsubscribe(ctx, subID)
				}
			})
		}

		select {
		case <-msg.InputChannel():
		case <-ctx.Done():
			cleanup()
			// Client disconnected before the stream went live — do not fall
			// through to phase 2 (would double-close and leak a subscription).
			return
		}

		subID = col.Subscribe(ctx, persistence.SubscriptionOptions{
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
				select {
				case docCh <- doc:
				default:
				}
				return nil
			},
		})

		<-ctx.Done()
		cleanup()
	}()

	return &abstract.Result{DocumentChannel: docCh}, nil
}

// StreamRegistration returns the system:audit:log:stream registration.
// The generator skips streaming annotations, so this is hand-written
// but uses dispatch.Handle for proper input binding.
func StreamRegistration(persist persistence.Persistence) []abstract.MessageRegistration {
	svc := &AuditService{persist: persist}
	return []abstract.MessageRegistration{
		{
			Name:        "system:audit:log:stream",
			Description: "Stream audit log entries in real-time",
			Intent:      abstract.Stream,
			Enabled:     true,
			Input: abstract.Input{
				Schema: dispatch.SchemaFromTypeWithTag[model.LogStreamInput]("input"),
			},
			Output:  dispatch.SchemaFromType[model.LogStreamOutput](),
			Handler: dispatch.Handle[model.LogStreamInput](svc.Stream),
		},
	}
}

// StreamPolicyBinding binds system:audit:log:stream to the administrator rule.
func StreamPolicyBinding() []policies.Binding {
	return []policies.Binding{
		{Name: "system:audit:log:stream", RuleKey: "administrator", Description: "Stream access logs in real-time"},
	}
}
