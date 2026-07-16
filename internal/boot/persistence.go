package boot

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8"
	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/go-anansi/v8/utils"
	events "github.com/asaidimu/go-events/v2"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/app/core"
)

type PersistenceManager struct {
	Anansi base.Persistence
	closer func()
}

func traceIDMetadataProvider() data.MetadataProviderConfig {
	return data.MetadataProviderConfig{
		Name: "trace_id_provider",
		Schema: &definition.NestedSchema{
			BaseSchema: definition.BaseSchema{
				Name: "trace_id_meta",
				Fields: map[definition.FieldId]definition.Field{
					"019f7a00-0001-7000-8000-000000000001": {
						Name:             "trace_id",
						FieldProperties: definition.FieldProperties{Type: definition.FieldTypeString},
					},
				},
			},
		},
		Provider: func(ctx context.Context, doc *data.Document) (map[string]any, error) {
			if traceID := core.GetTraceID(ctx); traceID != "" {
				return map[string]any{"trace_id": traceID}, nil
			}
			return nil, nil
		},
	}
}

func NewPersistenceManager(cfg *core.Config, logger *zap.Logger) (*PersistenceManager, error) {
	var p base.Persistence
	var icloser func()

	if cfg.PersistenceFactory != nil {
		setupCfg := anansi.SetupConfig{
			Logger: logger,
			DocumentFactoryConfig: data.DocumentFactoryConfig{
				Providers: []data.MetadataProviderConfig{traceIDMetadataProvider()},
				GlobalSanitizer: &data.FieldMaskConfig{
					DefaultPolicy: data.MaskPreserve,
					Fields: map[string]data.MaskedFieldPolicy{
						"hash": data.MaskRedact,
					},
					Patterns: []data.PatternRule{
						data.MustCompilePattern(`(?i)password`, data.MaskRedact),
						data.MustCompilePattern(`(?i)hash`, data.MaskRedact),
						data.MustCompilePattern(`(?i)secret`, data.MaskRedact),
						data.MustCompilePattern(`(?i)token`, data.MaskRedact),
						data.MustCompilePattern(`(?i)api[_-]?key`, data.MaskRedact),
						data.MustCompilePattern(`(?i)credential`, data.MaskRedact),
						data.MustCompilePattern(`(?i)auth`, data.MaskHash),
					},
				},
			},
		}

		var err error
		p, err = cfg.PersistenceFactory(&setupCfg)
		if err != nil {
			return nil, fmt.Errorf("persistence factory: %w", err)
		}
		icloser = func() {}
	} else {
		var interactor query.DatabaseInteractor

		if cfg.InteractorFactory != nil {
			var err error
			interactor, icloser, err = cfg.InteractorFactory(logger)
			if err != nil {
				return nil, fmt.Errorf("interactor factory: %w", err)
			}
		} else {
			db, err := NewDatabase(cfg, logger)
			if err != nil {
				return nil, fmt.Errorf("failed to create database: %w", err)
			}
			interactor = db.Interactor
			icloser = func() { _ = db.Close() }
		}

		eventBus, err := events.NewEventBus(events.DefaultConfig(cfg.DataDir, "persistence-events"))
		if err != nil {
			icloser()
			return nil, fmt.Errorf("failed to create event bus: %w", err)
		}

		bus := events.NewSimple[base.PersistenceEvent](eventBus, events.SimpleConfig{
			LiveOnly: true,
		})

		setupCfg := anansi.SetupConfig{
			Interactor: interactor,
			Logger:     logger,
			EventBus:   bus,
			DocumentFactoryConfig: data.DocumentFactoryConfig{
				Providers: []data.MetadataProviderConfig{traceIDMetadataProvider()},
				GlobalSanitizer: &data.FieldMaskConfig{
					DefaultPolicy: data.MaskPreserve,
					Fields: map[string]data.MaskedFieldPolicy{
						"hash": data.MaskRedact,
					},
					Patterns: []data.PatternRule{
						data.MustCompilePattern(`(?i)password`, data.MaskRedact),
						data.MustCompilePattern(`(?i)hash`, data.MaskRedact),
						data.MustCompilePattern(`(?i)secret`, data.MaskRedact),
						data.MustCompilePattern(`(?i)token`, data.MaskRedact),
						data.MustCompilePattern(`(?i)api[_-]?key`, data.MaskRedact),
						data.MustCompilePattern(`(?i)credential`, data.MaskRedact),
						data.MustCompilePattern(`(?i)auth`, data.MaskHash),
					},
				},
			},
			Schemas: nil,
		}

		p, err = anansi.Setup(setupCfg)
		if err != nil {
			icloser()
			return nil, fmt.Errorf("failed to setup Anansi: %w", err)
		}

		logger.Info("Persistence layer initialized — waiting for module schemas.")
	}

	sanitizationPolicyStore, err := utils.NewSanitizationPolicyStore(p, logger)
	if err != nil {
		icloser()
		return nil, fmt.Errorf("failed to setup sanitization: %w", err)
	}

	reg := data.GetSanitizationRegistry()
	reg.SetPersistence(sanitizationPolicyStore)

	if err := reg.LoadFromPersistence(context.Background()); err != nil {
		logger.Warn("Failed to load sanitization policies from persistence, using in-code defaults", zap.Error(err))
	}

	_ = reg.Register("_user_", &data.FieldMaskConfig{
		DefaultPolicy: data.MaskPreserve,
		Fields: map[string]data.MaskedFieldPolicy{
			"password": data.MaskRedact,
			"email":    data.MaskPreserve,
		},
	})
	_ = reg.Register("_api_key_", &data.FieldMaskConfig{
		DefaultPolicy: data.MaskPreserve,
		Fields: map[string]data.MaskedFieldPolicy{
			"hash": data.MaskRedact,
		},
	})

	return &PersistenceManager{
		Anansi: p,
		closer: icloser,
	}, nil
}

func (pm *PersistenceManager) Close() error {
	if pm.closer != nil {
		pm.closer()
	}
	return nil
}

func (pm *PersistenceManager) Collection(ctx context.Context, name string) (base.Collection, error) {
	return pm.Anansi.Collection(ctx, name)
}

func (pm *PersistenceManager) Persistence() base.Persistence {
	return pm.Anansi
}
