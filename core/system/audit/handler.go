// @note #cruft-20260821-025 observation open status=open priority=P3 tags=#cruft,#note : Old-style handler function in audit/handler.go
// @see #8uuufn
// No action needed — streaming handler for real-time log streaming, specialized use case.
//
// This file contains logStreamHandler — an old-style handler that returns
// abstract.MessageHandler directly. It uses a streaming pattern with
// DocumentChannel that is not supported by the generated registration system.
//
// This handler is used by the audit service for real-time log streaming,
// which is a specialized use case that cannot be handled by the generated
// registration pattern.
//
// Resolution: no action needed — this handler serves a specialized streaming
// purpose that is not covered by the generated registration system.
package audit

import (
	"context"
	"fmt"
	"sync"

	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"

	"github.com/asaidimu/hestia/core/abstract"
)

func logStreamHandler(persist base.Persistence) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		docCh := make(chan *document.Document, 64)

		col, err := persist.Collection(ctx, auditCollectionName)
		if err != nil {
			return nil, fmt.Errorf("get audit_log collection: %w", err)
		}

		go func() {
			select {
			case <-msg.InputChannel():
			case <-ctx.Done():
				close(docCh)
				return
			}

			var mu sync.Mutex
			closed := false

			subID := col.Subscribe(ctx, base.SubscriptionOptions{
				Event: base.DocumentCreateSuccess,
				Callback: func(_ context.Context, event base.PersistenceEvent) error {
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
					mu.Lock()
					defer mu.Unlock()
					if closed {
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

			mu.Lock()
			closed = true
			close(docCh)
			mu.Unlock()
			col.Unsubscribe(ctx, subID)
		}()

		return &abstract.Result{DocumentChannel: docCh}, nil
	}
}
