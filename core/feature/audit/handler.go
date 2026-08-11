package audit

import (
	"context"
	"fmt"
	"sync"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"

	"github.com/asaidimu/hestia/core/abstract"
)

func logStreamHandler(persist base.Persistence) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		docCh := make(chan data.Documenter, 64)

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
					doc := data.MustNewDocument(dataMap, context.Background())
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
