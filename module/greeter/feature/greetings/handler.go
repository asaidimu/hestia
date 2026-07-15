package greetings

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8/core/data"

	"github.com/asaidimu/hestia/internal/core"
	"github.com/asaidimu/hestia/internal/core/registration"
)

func NewCreateSalutationHandler(store *GreetingStore) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		body, _ := doc.GetOr("payload", nil).(map[string]any)
		phrase, _ := body["phrase"].(string)
		creator, _ := body["creator"].(string)

		if phrase == "" {
			return nil, fmt.Errorf("phrase is required")
		}
		if creator == "" {
			creator = "anonymous"
		}

		g, err := store.Create(ctx, phrase, creator)
		if err != nil {
			return nil, fmt.Errorf("create salutation: %w", err)
		}

		return &registration.Result{Document: data.MustNewDocument(map[string]any{
			"id":      g.ID,
			"phrase":  g.Phrase,
			"creator": g.Creator,
		})}, nil
	}
}

func NewGetSalutationHandler(store *GreetingStore) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		args, _ := doc.GetOr("arguments", nil).(map[string]any)
		id, _ := args["id"].(string)

		if id == "" {
			return nil, fmt.Errorf("id is required")
		}

		g, ok := store.Get(ctx, id)
		if !ok {
			return nil, fmt.Errorf("salutation %q not found", id)
		}

		return &registration.Result{Document: data.MustNewDocument(map[string]any{
			"id":      g.ID,
			"phrase":  g.Phrase,
			"creator": g.Creator,
		})}, nil
	}
}

func NewListSalutationsHandler(store *GreetingStore) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		salutes := store.List(ctx)
		items := make([]map[string]any, 0, len(salutes))
		for _, g := range salutes {
			items = append(items, map[string]any{
				"id":      g.ID,
				"phrase":  g.Phrase,
				"creator": g.Creator,
			})
		}
		return &registration.Result{Document: data.MustNewDocument(map[string]any{
			"salutations": items,
			"total":       len(items),
		})}, nil
	}
}

func NewGenerateGreetingHandler(store *GreetingStore) core.MessageHandler {
	return func(ctx context.Context, msg core.Message) (*registration.Result, error) {
		doc := msg.Input()
		body, _ := doc.GetOr("payload", nil).(map[string]any)
		name, _ := body["name"].(string)
		salutationID, _ := body["salutation_id"].(string)

		if name == "" {
			return nil, fmt.Errorf("name is required")
		}

		greeting := store.Generate(ctx, name, salutationID)
		return &registration.Result{
			Document: data.MustNewDocument(map[string]any{"greeting": greeting}),
		}, nil
	}
}
