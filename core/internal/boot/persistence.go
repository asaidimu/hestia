package boot

import (
	"context"
	"fmt"

	"github.com/asaidimu/go-anansi/v8"
	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	pevents "github.com/asaidimu/go-anansi/v8/core/persistence/events"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"github.com/asaidimu/go-anansi/v8/core/sanitize"
	"github.com/asaidimu/go-anansi/v8/utils"
	events "github.com/asaidimu/go-events/v2"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/internal/migrations"
	"github.com/asaidimu/hestia/core/runtime"
)

type PersistenceManager struct {
	Anansi base.Persistence
	closer func()
}

func docFactoryConfig() data.DocumentFactoryConfig {
	return data.DocumentFactoryConfig{
		Providers: migrations.MetadataProviderConfigs(),
	}
}

// sanitizeConfig intentionally registers masking rules per system collection
// using exact field names, and keeps the global policy preserve-only.
//
// Sanitization policies match by field NAME alone and masked values are
// always strings; the sanitizer does not know a field's declared type. A
// broad pattern (e.g. `(?i)auth` matching an `authors` array) therefore
// fails Document.Sanitize for the entire document ("cannot store sanitized
// value ... in array slot"), which silently empties query responses. See
// go-anansi devnote #sanitize-type-blindness. Because dynamic collections
// are user-defined, no global rule can be proven type-safe — never add
// Patterns here.
func sanitizeConfig() sanitize.Config {
	return sanitize.Config{
		Global: &sanitize.FieldMaskConfig{
			DefaultPolicy: sanitize.MaskPreserve,
		},
		Scoped: map[string]*sanitize.FieldMaskConfig{
			"_user_": {
				DefaultPolicy: sanitize.MaskPreserve,
				Fields: map[string]sanitize.MaskedFieldPolicy{
					"password":      sanitize.MaskRedact,
					"email":         sanitize.MaskPreserve,
					"token_version": sanitize.MaskPreserve,
				},
			},
			"_api_key_": {
				DefaultPolicy: sanitize.MaskPreserve,
				Fields: map[string]sanitize.MaskedFieldPolicy{
					"hash": sanitize.MaskRedact,
				},
			},
			"_access_log_": {
				DefaultPolicy: sanitize.MaskPreserve,
				Fields: map[string]sanitize.MaskedFieldPolicy{
					"integrity_hash": sanitize.MaskRedact,
				},
			},
		},
	}
}

func NewPersistenceManager(cfg *runtime.Config, logger *zap.Logger) (*PersistenceManager, error) {
	if err := sanitize.Configure(sanitizeConfig(), logger); err != nil {
		return nil, fmt.Errorf("configure sanitization: %w", err)
	}

	var p base.Persistence
	var icloser func()

	if cfg.PersistenceFactory != nil {
		var err error
		p, err = cfg.PersistenceFactory(&anansi.SetupConfig{
			Logger:                logger,
			DocumentFactoryConfig: docFactoryConfig(),
		})
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

		bus := pevents.NewGoEventsBusAdapter[base.PersistenceEvent](eventBus)

		p, err = anansi.Setup(anansi.SetupConfig{
			Interactor:            interactor,
			Logger:                logger,
			EventBus:              bus,
			DocumentFactoryConfig: docFactoryConfig(),
			Schemas:               nil,
		})

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

	reg := sanitize.Registry()
	reg.SetPersistence(sanitizationPolicyStore)

	if err := reg.LoadFromPersistence(context.Background()); err != nil {
		logger.Warn("Failed to load sanitization policies from persistence, using in-code defaults", zap.Error(err))
	}

	_ = reg.Register("_user_", &sanitize.FieldMaskConfig{
		DefaultPolicy: sanitize.MaskPreserve,
		Fields: map[string]sanitize.MaskedFieldPolicy{
			"password":      sanitize.MaskRedact,
			"email":         sanitize.MaskPreserve,
			"token_version": sanitize.MaskPreserve,
		},
	})
	_ = reg.Register("_api_key_", &sanitize.FieldMaskConfig{
		DefaultPolicy: sanitize.MaskPreserve,
		Fields: map[string]sanitize.MaskedFieldPolicy{
			"hash": sanitize.MaskRedact,
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
