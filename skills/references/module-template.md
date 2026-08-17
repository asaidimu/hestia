# Module template — copy this, don't reconstruct it

A module bundles features into an app. One module per app feature-area (or one
per app — `boot.ProjectName` and generated model singletons mean **one hestia
app per process**, so don't try to stack two apps).

Verified against `core/abstract/module.go`, `core/system/module.go`
(SystemModule), `core/internal/boot/app.go` (`RegisterModules`), and
`core/hestia.go` (`Setup`/`SetupConfig`).

## The `Module` interface

```go
type Module interface {
	Name() string
	Setup(ctx context.Context, rt abstract.Container) error
	Capabilities(rt abstract.Container) ([]Capability, error) // Name + Messages []MessageRegistration
	Dependencies() []string                // module names that must init first
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health(ctx context.Context) any
}
```

## A minimal module

```go
package mymod

import (
	"context"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime/module"
)

type Module struct {
	module.BaseModule // provides default Dependencies/Start/Stop/Health
}

func (m *Module) Name() string { return "mymod" }

// Setup runs in the first boot phase, in dependency order, with the shared
// runtime container. Register your service providers here; resolve the base
// providers (persistence, logger, dispatcher) via abstract.MustResolve[T](rt).
func (m *Module) Setup(ctx context.Context, rt abstract.Container) error {
	// persist := abstract.MustResolve[base.Persistence](rt)
	return myfeature.Register(rt) // register the feature's service provider
}

// Capabilities runs in the second boot phase, after the container has been
// sealed. Resolve your services and return their message registrations.
func (m *Module) Capabilities(rt abstract.Container) ([]abstract.Capability, error) {
	regs, err := myfeature.Registrations(rt)
	if err != nil {
		return nil, err
	}
	return []abstract.Capability{
		{Name: "mymod", Messages: regs},
	}, nil
}
```

`RegisterModules` (see `core/internal/boot/app.go`) topological-sorts by
`Dependencies`, then in phase 1 calls `Setup` on each (providers registered
into the shared container, seeded with persistence/logger/dispatcher), in phase
2 seals the container with `Rebuild`, then in phase 3 calls each module's
`Capabilities` to collect registrations, validates each name (must be 4
segments) and installs the handlers into the `LocalDispatcher`. This is why
features expose `Register(rt)` + `Registrations(rt)` — the module's `Setup`
registers their providers and `Capabilities` resolves them.

## Wiring the app (`cmd/<app>/main.go`)

The extension points (`Migrate`, `DispatcherChainFunc`, `BuildInterfaces`,
`OnBootstrapped`, `OnReset`, `PersistenceFactory`) are **`SetupConfig` fields**,
not Module methods. Wire them here:

```go
package main

import (
	"fmt"
	"os"

	"github.com/asaidimu/go-anansi/v8/core/persistence/base"

	hestia "github.com/asaidimu/hestia/core"
	"github.com/asaidimu/hestia/core/runtime"
)

func main() {
	app, err := hestia.Setup(hestia.SetupConfig{
		SessionSecret: "s3cret",
		Modules:       []hestia.Module{&mymod.Module{}}, // or RegisterModules after Setup
		Migrate: func(ctx context.Context, p base.Persistence) error {
			return mydata.Migrate(ctx, p) // your collection/migrations
		},
		DispatcherChainFunc: func(chain abstract.ChainEditor) {
			// insert links AFTER "secure" — never before it
		},
		OnBootstrapped: func() { fmt.Println("bootstrapped") },
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := app.Start(); err != nil { // Start, not Run
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer app.Close()
	select {}
}
```

Notes verified from `core/hestia.go`:
- `SessionSecret` is **required** (or `SESSION_SECRET` env).
- `Setup` runs boot first (migrations + SystemModule), **then** `cfg.Migrate`
  with `application.Persistence()`, then registers `cfg.Modules`, then builds
  interfaces. Default interfaces are HTTP (`/api`, fasthttp) + CLI.
- `SetupConfig.Bootstrap *bool` — set `false` to skip the boot phase entirely
  (returns a wired but un-booted app).
- `RegisterModules(...)` on `*Application` is the post-Setup equivalent.

## Reaching built-in features from your module

Your `Setup` receives the shared runtime container. For the typed built-in
models/services, resolve them via `abstract.MustResolve[T](rt)` (persistence,
logger, dispatcher are pre-seeded; the system providers are seeded by the
system module's `Setup`). For system-wide state (admin user ID, credentials
provider, bootstrapped flag), go through `app.SystemModule()` (after boot):
`UserModel()`, `CredentialsProvider()`, `AdminUserID()`, `AdminEmail()`,
`Bootstrapped()`. To *invoke* built-in functionality (e.g. send a
notification), dispatch a message rather than importing the model and writing
directly — the dispatcher enforces your policies:

```go
msg := dispatch.NewMessage("system:notifications:notification:create", ctx, inputDoc)
result, err := app.Dispatcher().Dispatch(ctx, msg)
```

## Your feature's files

Same 5-file layout as any `core/system/<feature>` — see
`references/feature-template.md`.