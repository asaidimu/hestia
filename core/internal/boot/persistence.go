package boot

import (
	"context"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/sanitize"
	"github.com/asaidimu/go-anansi/v8/utils"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/internal/migrations"
	"github.com/asaidimu/hestia/core/runtime"
)

var ErrPersistenceFactoryMissing = common.NewSystemError(
	"ERR_PERSISTENCE_FACTORY_MISSING",
	"no PersistenceFactory configured: hestia core ships no database driver; "+
		"pass an explicit backend to Setup, e.g. PersistenceFactory: "+
		"sqlite.Default(dbPath) from github.com/asaidimu/hestia/core/persistence/sqlite, "+
		"or provide a custom runtime.PersistenceFactory")

type PersistenceManager struct {
	Anansi base.Persistence
	closer func()
}

func docFactoryConfig() data.DocumentFactoryConfig {
	return data.DocumentFactoryConfig{
		Providers: migrations.MetadataProviderConfigs(),
	}
}

// sanitizeConfig returns the global sanitization configuration. Scoped rules
// are now registered per-feature via allSanitizationRules; this function only
// sets the global default policy.
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
	}
}

func NewPersistenceManager(cfg *runtime.Config, logger *zap.Logger) (*PersistenceManager, error) {
	if err := sanitize.Configure(sanitizeConfig(), logger); err != nil {
		return nil, common.SystemErrorFrom(err, "ERR_SANITIZE_CONFIGURE").
			WithOperation("NewPersistenceManager")
	}

	if err := data.ConfigureDocumentFactory(docFactoryConfig(), logger); err != nil {
		return nil, common.SystemErrorFrom(err, "ERR_DOCUMENT_FACTORY_CONFIGURE").
			WithOperation("NewPersistenceManager")
	}

	if cfg.PersistenceFactory == nil {
		return nil, ErrPersistenceFactoryMissing.WithOperation("NewPersistenceManager")
	}

	p, closer, err := cfg.PersistenceFactory(runtime.PersistenceDeps{
		Logger:  logger,
		DataDir: cfg.DataDir,
		DBPath:  cfg.DBPath,
	})
	if err != nil {
		return nil, common.SystemErrorFrom(err, "ERR_PERSISTENCE_FACTORY").
			WithOperation("NewPersistenceManager")
	}
	if closer == nil {
		closer = func() {}
	}

	sanitizationPolicyStore, err := utils.NewSanitizationPolicyStore(p, logger)
	if err != nil {
		closer()
		return nil, common.SystemErrorFrom(err, "ERR_SANITIZATION_STORE").
			WithOperation("NewPersistenceManager")
	}

	reg := sanitize.Registry()
	reg.SetPersistence(sanitizationPolicyStore)

	if err := reg.LoadFromPersistence(context.Background()); err != nil {
		logger.Warn("Failed to load sanitization policies from persistence, using in-code defaults", zap.Error(err))
	}

	return &PersistenceManager{
		Anansi: p,
		closer: closer,
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
