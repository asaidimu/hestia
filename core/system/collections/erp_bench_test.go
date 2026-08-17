package collections_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync/atomic"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"github.com/asaidimu/go-anansi/v8/core/schema/definition"
	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/system/collections"
	"github.com/asaidimu/hestia/core/internal/testutil"
	"github.com/asaidimu/hestia/core/runtime"
	"github.com/asaidimu/hestia/core/runtime/audit"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
)

const erpCollection = "erp_orders"

// These benchmarks measure ERP-shaped workloads — not framework hot loops:
// concurrent document CRUD through the full production chain (bootstrap ->
// secure -> audit), batch ingestion, and report-shaped paginated queries.
// The framework's fixed per-request cost (~single-digit µs) is visible here
// only as the delta; the numbers are dominated by the persistence layer.

type benchPersister struct{ count atomic.Int64 }

func (p *benchPersister) Insert(_ context.Context, _ audit.AuditEntry) error {
	p.count.Add(1)
	return nil
}

type benchNoop struct{}

func (benchNoop) Send(abstract.Message) (*abstract.Result, error) { return &abstract.Result{}, nil }

type benchBootstrapRegistry struct{}

func (benchBootstrapRegistry) IsHandlerBootstrapSafe(string) bool { return true }

func compileErpRule(ac iam.AccessController, expr string) iam.FunctionRule {
	fn, err := ac.CompileCELRule(expr)
	if err != nil {
		panic(err)
	}
	return fn
}

func erpAdminCtx(tenant string) context.Context {
	ctx := iam.WithIdentity(context.Background(), iam.Identity{
		Permissions: []string{"administrator"},
		Properties:  map[string]any{
			"user_id":     "u1",
			"permissions": []string{"administrator"},
			"token_type":  "access",
		},
	})
	return runtimecontext.ContextWithTenantID(ctx, tenant)
}

func erpSchema() *definition.Schema {
	return &definition.Schema{
		Version: common.NewVersionFromUint64(1),
		BaseSchema: definition.BaseSchema{
			Name: erpCollection,
			Fields: map[definition.FieldId]definition.Field{
				"order_no": {Name: "order_no", FieldProperties: definition.FieldProperties{Type: definition.FieldTypeString}},
				"amount":   {Name: "amount", FieldProperties: definition.FieldProperties{Type: definition.FieldTypeNumber}},
				"status":   {Name: "status", FieldProperties: definition.FieldProperties{Type: definition.FieldTypeString}},
			},
		},
	}
}

// erpChain builds the production dispatch chain (bootstrap -> secure -> audit)
// over a LocalDispatcher with the document handlers registered for a business
// collection. Returns the chain plus a counting audit persister.
func erpChain(b *testing.B) (abstract.Dispatcher, *benchPersister) {
	b.Helper()
	ctx := context.Background()
	p := testutil.NewPersistenceTB(b)

	if _, err := p.CreateCollection(ctx, erpSchema()); err != nil {
		b.Fatalf("create %s collection: %v", erpCollection, err)
	}

	base := runtime.NewLocalDispatcher()
	regs := []struct {
		name string
		h    abstract.MessageHandler
	}{
		{"erp:orders:order:create", collections.NewDocumentCreateHandler(p)},
		{"erp:orders:order:read", collections.NewDocumentGetHandler(p)},
		{"erp:orders:order:update", collections.NewDocumentUpdateHandler(p)},
		{"erp:orders:order:delete", collections.NewDocumentDeleteHandler(p)},
		{"erp:orders:order:query", collections.NewCollectionQueryHandler(p)},
	}
	permMgr := runtime.NewMapPermissionManager()
	for _, r := range regs {
		if err := base.RegisterHandler(r.name, r.h, abstract.HandlerInfo{Name: r.name, Enabled: true}); err != nil {
			b.Fatalf("RegisterHandler %s: %v", r.name, err)
		}
		permMgr.RegisterScope(r.name, "administrator", "")
	}
	ac := iam.CreateAccessController(iam.AccessControllerOptions{}, slog.New(slog.NewTextHandler(discarder{}, nil)))
	ac.LoadRules(iam.FunctionRuleSet{
		"administrator": compileErpRule(ac, "identity != null && 'administrator' in identity.permissions"),
	})

	persister := &benchPersister{}
	secure := runtime.NewSecureDispatcher(base, permMgr, ac)
	auditDisp := runtime.NewAuditDispatcher(secure, persister)
	bootstrap := runtime.NewBootstrapDispatcher(benchNoop{}, benchBootstrapRegistry{}, func() bool { return true })
	b.Cleanup(func() { auditDisp.Close() })

	disp := runtime.NewDispatcherChain(
		runtime.LinkEntry{Name: "bootstrap", Link: bootstrap},
		runtime.LinkEntry{Name: "secure", Link: secure},
		runtime.LinkEntry{Name: "audit", Link: auditDisp},
	).Build(base)

	return disp, persister
}

func erpCreateMsg(id string, ctx context.Context) abstract.Message {
	doc := data.MustNewDocument(map[string]any{
		"arguments": map[string]any{"name": erpCollection},
		"payload":   map[string]any{"order_no": id, "amount": float64(99.5), "status": "open"},
	}, ctx)
	return dispatch.NewMessage("erp:orders:order:create", ctx, doc)
}

func erpReadMsg(id string, ctx context.Context) abstract.Message {
	doc := data.MustNewDocument(map[string]any{
		"arguments": map[string]any{"name": erpCollection, "doc_id": id},
	}, ctx)
	return dispatch.NewMessage("erp:orders:order:read", ctx, doc)
}

// BenchmarkERP_ConcurrentDocCreate measures concurrent order ingestion (the
// ERP write hot path) through the full chain. SQLite serializes writers, so
// this captures the real ERP ceiling — not just the framework overhead.
func BenchmarkERP_ConcurrentDocCreate(b *testing.B) {
	disp, _ := erpChain(b)
	ctx := erpAdminCtx("tenant-a")
	var counter atomic.Int64

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := fmt.Sprintf("ord-%d", counter.Add(1))
			if _, err := disp.Send(erpCreateMsg(id, ctx)); err != nil {
				b.Error(err)
				return
			}
		}
	})
}

// BenchmarkERP_ConcurrentDocRead measures concurrent order lookups — ERP reads
// are the majority of traffic and can parallelize on SQLite.
func BenchmarkERP_ConcurrentDocRead(b *testing.B) {
	disp, _ := erpChain(b)
	ctx := erpAdminCtx("tenant-a")

	const seeded = 200
	for i := 0; i < seeded; i++ {
		if _, err := disp.Send(erpCreateMsg(fmt.Sprintf("ord-%d", i), ctx)); err != nil {
			b.Fatalf("seed: %v", err)
		}
	}
	var counter atomic.Int64

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			n := counter.Add(1) % seeded
			if _, err := disp.Send(erpReadMsg(fmt.Sprintf("ord-%d", n), ctx)); err != nil {
				b.Error(err)
				return
			}
		}
	})
}

// BenchmarkERP_BatchCreate_ThroughChain measures bulk ingestion with the full
// chain (audit on) — the month-end-close style batch path.
func BenchmarkERP_BatchCreate_ThroughChain(b *testing.B) {
	disp, persister := erpChain(b)
	ctx := erpAdminCtx("tenant-a")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := disp.Send(erpCreateMsg(fmt.Sprintf("ord-%d", i), ctx)); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	b.ReportMetric(float64(persister.count.Load()), "audit-entries")
}

// BenchmarkERP_BatchCreate_DirectHandler is the no-chain baseline (no authz,
// no audit) so the framework+audit delta on the write path is measurable.
func BenchmarkERP_BatchCreate_DirectHandler(b *testing.B) {
	ctx := erpAdminCtx("tenant-a")
	p := testutil.NewPersistenceTB(b)
	if _, err := p.CreateCollection(ctx, erpSchema()); err != nil {
		b.Fatalf("create collection: %v", err)
	}
	h := collections.NewDocumentCreateHandler(p)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := h(ctx, erpCreateMsg(fmt.Sprintf("ord-%d", i), ctx)); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkERP_Query_Paginated measures report-shaped reads: filtered, sorted,
// offset-paginated queries walking a large dataset.
func BenchmarkERP_Query_Paginated(b *testing.B) {
	ctx := erpAdminCtx("tenant-a")
	p := testutil.NewPersistenceTB(b)
	if _, err := p.CreateCollection(ctx, erpSchema()); err != nil {
		b.Fatalf("create collection: %v", err)
	}
	h := collections.NewCollectionQueryHandler(p)

	// Seed a sizeable dataset once (outside the measured loop).
	const datasetSize = 5000
	for i := 0; i < datasetSize; i++ {
		if _, err := collections.NewDocumentCreateHandler(p)(ctx, erpCreateMsg(fmt.Sprintf("ord-%d", i), ctx)); err != nil {
			b.Fatalf("seed: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q := query.NewQueryBuilder().
			Where("status").Eq("open").
			OrderByAsc("order_no").
			Limit(100).
			Offset(i % 50).
			Build()
		body, err := json.Marshal(q)
		if err != nil {
			b.Fatal(err)
		}
		doc := data.MustNewDocument(map[string]any{
			"arguments": map[string]any{"name": erpCollection},
			"payload":   json.RawMessage(body),
		}, ctx)
		msg := dispatch.NewMessage("erp:orders:order:query", ctx, doc)
		if _, err := h(ctx, msg); err != nil {
			b.Fatal(err)
		}
	}
}

type discarder struct{}

func (discarder) Write(p []byte) (int, error) { return len(p), nil }