# Feature template — copy this, don't reconstruct it

Distilled from `core/system/apikeys/` (the canonical migrated reference). Copy
these shapes verbatim and rename. If you only remember one thing, remember the
**layout**: a feature is five Go files plus a schema, all under
`core/system/<feature>/`:

```
core/system/<feature>/
├── model/
│   ├── <name>.schema.json        # the collection schema + projections
│   ├── <name>.schema.model.go    # GENERATED — never hand-edit
│   ├── data_transfer_objects.go  # input/output DTOs + schema builders
│   └── <feature>_utils.go        # domain methods on the generated model
├── service.go                    # the service struct + New<Feature>Service(rt)
├── handler.go                    # message handler methods (annotate @hestia.register)
├── feature.go                    # Register(rt) + Registrations(rt) → []MessageRegistration
├── provider.go                   # policy bindings (if any)
├── handler_test.go               # handler tests
└── model/<name>_test.go          # domain method tests
```

## 1. Schema — `model/<name>.schema.json`

Plain JSON, package `model`. Keep the UUID field IDs; never re-key them. Add
fields with `"id": "<name>"` matching the field name. The `_id_`, `_metadata_`,
and `_metadata_` sub-schema blocks are boilerplate — copy them from an existing
feature.

Projections go in `metadata.projections`. Every projection gets input tags:
`payload.{name}` for payload fields, `arguments.<arg_name>` for path-arg fields.
**Exclude secrets from the public/output projection** (apikeys excludes `hash`
from `APIKeyPublic`).

Required vs optional in a projection's `fields`:
- `required` — must be present in the request
- `optional` — may be present
- `include` + `optional`/`required` together, or `exclude` for the output
  projection.

`anansi codegen golang` turns this into the typed model + projections.
**Never hand-edit the generated `.model.go`.** See the `anansi` skill for the
full schema workflow (also in `core/system/users`).

## 2. DTOs — `model/data_transfer_objects.go`

```go
package model

import (
	"github.com/asaidimu/go-anansi/v8/core/document"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"

	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

// Small struct for argument-only inputs.
type WidgetGetInput struct {
	WidgetID string `input:"arguments.widget_id"`
}

// Output envelope: single-document response.
type WidgetOutput struct {
	Document WidgetPublic `anansi:"document"`
}

// Output envelope: list response.
type WidgetListOutput struct {
	Documents []WidgetPublic `anansi:"documents"`
}

func WidgetGetInputSchema() *definition.Schema {
	return dispatch.SchemaFromTypeWithTag[WidgetGetInput]("input")
}
func WidgetCreateInputSchema() *definition.Schema {
	return dispatch.SchemaFromTypeWithTag[WidgetCreate]("input", true)
}
func WidgetUpdateInputSchema() *definition.Schema {
	return dispatch.SchemaFromTypeWithTag[WidgetUpdate]("input", true)
}
func WidgetOutputSchema() *definition.Schema {
	return dispatch.SchemaFromType[WidgetOutput]()
}
func WidgetListOutputSchema() *definition.Schema {
	return dispatch.SchemaFromType[WidgetListOutput]()
}
```

- Arg-only inputs: `SchemaFromTypeWithTag[T]("input")`.
- Payload-bearing create/update: pass `true` (`SchemaFromTypeWithTag[T]("input", true)`).
- Output envelopes use `anansi:"document"` / `anansi:"documents"` tags — never
  call `SchemaFromTypeWithTag` on outputs.

## 3. Domain methods — `model/<feature>_utils.go`

Generated model type is `System<Feature>` (singular), collection
`System<Features>`. Methods on the collection receiver (`m *SystemWidgets`):

```go
func (m *SystemWidgets) List(ctx context.Context, userID string) ([]*SystemWidget, error) {
	q := query.NewQueryBuilder().Where("userId").Eq(userID).Build()
	return m.Read(ctx, &q)
}

func (m *SystemWidgets) Get(ctx context.Context, widgetID, userID string) (*SystemWidget, error) {
	q := query.NewQueryBuilder().
		Where(data.DocumentIDField).Eq(widgetID).
		Where("userId").Eq(userID).
		Limit(1).Build()
	widgets, err := m.Read(ctx, &q)
	if err != nil {
		return nil, err
	}
	if len(widgets) == 0 {
		return nil, fmt.Errorf("widget not found")
	}
	return widgets[0], nil
}

func (m *SystemWidgets) Create(ctx context.Context, userID string, req *WidgetCreate) (*SystemWidget, error) {
	doc := document.New(&SystemWidget{ /* ... */ UserID: userID})
	return m.ModelCollection.Create(ctx, doc)
}

func (m *SystemWidgets) UpdateWidget(ctx context.Context, widgetID, userID string, req *WidgetUpdate) (*SystemWidget, error) {
	if _, err := m.Get(ctx, widgetID, userID); err != nil {
		return nil, err
	}
	return m.UpdateFrom[*WidgetUpdate, *SystemWidget](ctx, widgetID, req)
}

func (m *SystemWidgets) Delete(ctx context.Context, widgetID, userID string) error {
	if _, err := m.Get(ctx, widgetID, userID); err != nil {
		return err
	}
	return m.DeleteByID(ctx, widgetID)
}
```

Query DSL (from go-anansi): `query.NewQueryBuilder().Where("field").Eq(v).Limit(n).Build()`,
plus `OrderBy`/`Offset`. No joins, no aggregates, no text search. Ownership
scoping (`Where("userId")`) is the authorization boundary in the data layer.

## 4. Handler — `handler.go`

```go
package widgets

func NewGetWidgetHandler(model *model.SystemWidgets) abstract.MessageHandler {
	return func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
		var input model.WidgetGetInput
		if err := msg.Input().BindToTag(&input, "input"); err != nil {
			return nil, err
		}
		w, err := model.Get(ctx, input.WidgetID, userIDFrom(ctx, msg))
		if err != nil {
			return nil, err
		}
		doc, err := w.Document()
		if err != nil {
			return nil, err
		}
		return &abstract.Result{Document: doc}, nil
	}
}

func userIDFrom(ctx context.Context, msg abstract.Message) string {
	if claims, ok := runtimecontext.ClaimsFromContext(ctx); ok {
		return claims.UserID
	}
	var input model.WidgetListInput
	if err := msg.Input().BindToTag(&input, "input"); err != nil {
		return ""
	}
	return input.UserID
}
```

Read the claims with `runtimecontext.ClaimsFromContext` — the fallback to a
bound `user_id` argument is what lets admin/system callers pass an explicit
owner. Never trust a user-supplied `user_id` over the claims; if both are
present the claims win.

## 5. Registrations — `feature.go`

Services are registered into the shared runtime container and resolved in
`Registrations`. Message handlers are methods on the service struct (see §4b);
the generated `registrations.go` (from `hestia service generate`) binds them via
`dispatch.Handle*` adapters. The hand-written contract is:

```go
package widgets

import (
	"github.com/asaidimu/hestia/core/abstract"
)

// Register registers the WidgetsService provider into the shared container.
func Register(rt abstract.Container) error {
	return abstract.Register[*WidgetsService](rt, func(c abstract.Container) (*WidgetsService, error) {
		return NewWidgetsService(c)
	})
}

// Registrations resolves the service and returns its message registrations.
func Registrations(rt abstract.Container) ([]abstract.MessageRegistration, error) {
	s := abstract.MustResolve[*WidgetsService](rt)
	return []abstract.MessageRegistration{
		{
			Name: "system:widgets:widget:list", Handler: dispatch.HandleDocuments[WidgetListInput](s.List),
			Description: "List widgets", Enabled: true, Intent: abstract.Read,
			Input:  abstract.Input{Schema: dispatch.SchemaFromTypeWithTag[WidgetListInput]("input")},
			Output: dispatch.SchemaFromType[WidgetPublic](),
		},
		{
			Name: "system:widgets:widget:create", Handler: dispatch.HandleDocument[WidgetCreateInput, *Widget](s.Create),
			Description: "Create a widget", Enabled: true, Intent: abstract.Create,
			Input:  abstract.Input{Schema: dispatch.SchemaFromTypeWithTag[WidgetCreateInput]("input")},
			Output: dispatch.SchemaFromType[WidgetPublic](),
		},
	}, nil
}
```

Handlers annotate their service method with `@hestia.register(...)` and `hestia
service generate` emits `registrations.go`/`policies.go` — the manual table
above is only the shape when writing by hand. Message names are exactly
`module:feature:scope:action`. `Intent` drives the HTTP verb;
`ResourceIDField` shapes the path (`/module/feature/scope/action/{widget_id}`).

## 6. Policy bindings — `provider.go`

```go
func Provider() abstract.Provider {
	return abstract.Provider{
		Feature:         "widgets",
		PolicyBindings:  []policies.Binding{
			{Name: "manage:widgets", Description: "Create, update, and delete widgets"},
		},
		DefaultRuleKey:  "authenticated",
		RuleKey:         "widgets",
	}
}
```

See `core/system/apikeys/policies.go` (or `core/system/policies/model.go` for
the `Binding` shape) to verify field names before using. **Bind by name in the
provider; enable by assigning the rule in the database** — rule changes apply
immediately, no cache.

## 7. Test — `handler_test.go`

```go
func testModel(t *testing.T) *widgetsmodel.SystemWidgets {
	t.Helper()
	widgetsmodel.DangerouslyResetSystemWidgetsModel()
	m, err := widgetsmodel.InitSystemWidgetsModel(testutil.NewPersistence(t), zap.NewNop())
	if err != nil {
		t.Fatalf("InitSystemWidgetsModel: %v", err)
	}
	return m
}

func testMsg(name string, input data.Documenter) abstract.Message {
	return dispatch.NewMessage(name, context.Background(), input)
}

// authCtx wraps a claims-carrying context for authed handler tests.
func authCtx(t *testing.T, userID string) context.Context {
	return runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{UserID: userID})
}
```

Patterns:
- `testutil.NewPersistence(t)` gives an isolated DB; **always reset the model
  singleton first** via the generated `DangerouslyReset<Model>()`.
- Input docs are built with `testutil.InputDoc(t, <InputSchema>(), \`{"arguments": {...}}\`)`
  or `\`{"payload": {...}}\`` — matching the tag paths.
- Assert on `result.Document.GetString("field")` / `result.Documents`.
- Delete-test asserts the follow-up `Get` fails.