package runtime

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/base"
	"github.com/asaidimu/go-anansi/v8/core/persistence/collection"
	"github.com/asaidimu/go-anansi/v8/core/query"
	"github.com/asaidimu/go-iam/v2/iam"

	"github.com/asaidimu/hestia/core/internal/testutil"
)

// ruleDocProcessor compiles CEL expressions from _iam_rule_ documents into
// iam.FunctionRule values suitable for caching in a LiveCollection.
type ruleDocProcessor struct {
	ac iam.AccessController
}

func (p *ruleDocProcessor) Compile(ctx context.Context, doc *data.Document) (iam.FunctionRule, error) {
	expr, _ := doc.GetString("expression")
	if expr == "" {
		return func(req iam.AccessRequest) bool { return true }, nil
	}
	return p.ac.CompileCELRule(expr)
}

func (p *ruleDocProcessor) CloneState(fn iam.FunctionRule) (iam.FunctionRule, error) {
	return fn, nil
}

// writeableRuleStore captures the full LiveCollection API including DB-write
// methods (CreateOne, Update, Delete) that are not part of the LiveCollection
// interface but exist on the concrete liveRepository type.
type writeableRuleStore interface {
	collection.LiveCollection[iam.FunctionRule]
	CreateOne(ctx context.Context, doc *data.Document) (base.CreateResult, error)
	Update(ctx context.Context, params *base.CollectionUpdate) (*base.ReadResult, error)
}

func newRuleStore(t *testing.T, p base.Persistence, ac iam.AccessController) writeableRuleStore {
	t.Helper()
	ctx := context.Background()

	ruleColl, err := p.Collection(ctx, "_iam_rule_")
	if err != nil {
		t.Fatalf("getting _iam_rule_ collection: %v", err)
	}

	// A separate temporary AccessController for CEL compilation inside the
	// DocumentProcessor, to avoid circular dependency (the live collection
	// needs a compiler, and the AccessController uses the live collection).
	tmpAC := iam.CreateAccessController(iam.AccessControllerOptions{},
		slog.New(slog.NewTextHandler(discarder{}, nil)))

	live, err := collection.NewLiveRepository(ctx, collection.LiveRepositoryOptions[iam.FunctionRule]{
		Collection: ruleColl,
		Processor:  &ruleDocProcessor{ac: tmpAC},
		QueryKey:   "name",
		Active:     false,
	})
	if err != nil {
		t.Fatalf("NewLiveRepository: %v", err)
	}
	t.Cleanup(func() { live.Close() })

	ac.LoadRules(live)

	// Type-assert to gain access to CreateOne and Update on the concrete
	// liveRepository type.
	ws, ok := live.(writeableRuleStore)
	if !ok {
		t.Fatal("LiveCollection does not implement writeableRuleStore")
	}
	return ws
}

func TestLiveCollectionRuleStore_GetSet(t *testing.T) {
	// A LiveCollection[iam.FunctionRule] can be used as the RuleSet for an
	// AccessController. Rules set via Set() must be immediately resolvable
	// via Can().
	p := testutil.NewPersistence(t)
	ac := iam.CreateAccessController(iam.AccessControllerOptions{},
		slog.New(slog.NewTextHandler(discarder{}, nil)))
	live := newRuleStore(t, p, ac)

	can := ac.Can(adminContext(), "test_rule", nil, nil)
	if can {
		t.Fatal("expected Can()=false for unregistered rule, got true")
	}

	live.Set("test_rule", func(req iam.AccessRequest) bool { return true })

	if !ac.Can(anonymousContext(), "test_rule", nil, nil) {
		t.Fatal("expected Can()=true after Set, got false")
	}
}

func TestLiveCollectionRuleStore_UnsetRemoves(t *testing.T) {
	p := testutil.NewPersistence(t)
	ac := iam.CreateAccessController(iam.AccessControllerOptions{},
		slog.New(slog.NewTextHandler(discarder{}, nil)))
	live := newRuleStore(t, p, ac)

	live.Set("temp_rule", func(req iam.AccessRequest) bool { return true })
	if !ac.Can(anonymousContext(), "temp_rule", nil, nil) {
		t.Fatal("expected Can()=true after Set")
	}

	live.Unset("temp_rule")
	if ac.Can(anonymousContext(), "temp_rule", nil, nil) {
		t.Fatal("expected Can()=false after Unset, got true")
	}
}

func TestLiveCollectionRuleStore_AutoRefreshOnCreate(t *testing.T) {
	// A rule upserted via the LiveCollection's CreateOne (DB write + compile +
	// cache update) must be immediately visible to the AccessController
	// without a manual Reload call.
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	ac := iam.CreateAccessController(iam.AccessControllerOptions{},
		slog.New(slog.NewTextHandler(discarder{}, nil)))
	live := newRuleStore(t, p, ac)

	if ac.Can(adminContext(), "auto_rule", nil, nil) {
		t.Fatal("expected Can()=false before rule exists")
	}

	doc := data.MustNewDocument(map[string]any{
		"name":       "auto_rule",
		"ruleType":   "simple",
		"syntax":     "cel",
		"expression": "true",
	}, ctx)
	_, err := live.CreateOne(ctx, doc)
	if err != nil {
		t.Fatalf("CreateOne: %v", err)
	}

	if !ac.Can(anonymousContext(), "auto_rule", nil, nil) {
		t.Fatal("expected Can()=true after CreateOne (auto-refresh), got false")
	}
}

func TestLiveCollectionRuleStore_AutoRefreshOnUpdate(t *testing.T) {
	// Updating a rule via the LiveCollection must immediately reflect the
	// new expression without manual reload.
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	ac := iam.CreateAccessController(iam.AccessControllerOptions{},
		slog.New(slog.NewTextHandler(discarder{}, nil)))
	live := newRuleStore(t, p, ac)

	doc := data.MustNewDocument(map[string]any{
		"name":       "update_rule",
		"ruleType":   "simple",
		"syntax":     "cel",
		"expression": "false",
	}, ctx)
	_, err := live.CreateOne(ctx, doc)
	if err != nil {
		t.Fatalf("CreateOne: %v", err)
	}

	if ac.Can(adminContext(), "update_rule", nil, nil) {
		t.Fatal("expected Can()=false for rule with expression 'false'")
	}

	_, err = live.Update(ctx, &base.CollectionUpdate{
		Set: data.Patch(map[string]any{
			"expression": "true",
		}).Document(ctx),
		Filter: &query.QueryFilter{
			Condition: &query.FilterCondition{
				Field:    "name",
				Operator: query.ComparisonOperatorEq,
				Value:    query.FilterValue{StringVal: strPtr("update_rule")},
			},
		},
		ReturnDocument: true,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if !ac.Can(adminContext(), "update_rule", nil, nil) {
		t.Fatal("expected Can()=true after Update changed expression to 'true'")
	}
}

func TestLiveCollectionRuleStore_GoDefaultsCoexist(t *testing.T) {
	// Go default rules set via Set() must coexist with DB-backed rules
	// created via CreateOne.
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	ac := iam.CreateAccessController(iam.AccessControllerOptions{},
		slog.New(slog.NewTextHandler(discarder{}, nil)))
	live := newRuleStore(t, p, ac)

	live.Set("go_public", func(req iam.AccessRequest) bool { return true })
	live.Set("go_admin", func(req iam.AccessRequest) bool {
		ident, _ := req.Identity.(map[string]any)
		perms, _ := ident["permissions"].([]string)
		for _, p := range perms {
			if p == "administrator" {
				return true
			}
		}
		return false
	})

	doc := data.MustNewDocument(map[string]any{
		"name":       "cel_allow",
		"ruleType":   "simple",
		"syntax":     "cel",
		"expression": "'administrator' in identity.permissions",
	}, ctx)
	_, err := live.CreateOne(ctx, doc)
	if err != nil {
		t.Fatalf("CreateOne: %v", err)
	}

	if !ac.Can(anonymousContext(), "go_public", nil, nil) {
		t.Fatal("expected Can()=true for go_public (anonymous allowed)")
	}
	if ac.Can(anonymousContext(), "go_admin", nil, nil) {
		t.Fatal("expected Can()=false for go_admin (anonymous denied)")
	}
	if !ac.Can(adminContext(), "go_admin", nil, nil) {
		t.Fatal("expected Can()=true for go_admin (admin allowed)")
	}
	if !ac.Can(adminContext(), "cel_allow", nil, nil) {
		t.Fatal("expected Can()=true for cel_allow (admin allowed via CEL)")
	}
}

func TestSecureDispatcher_AutoRefreshedRule(t *testing.T) {
	// Full dispatch chain: a rule upserted via LiveCollection must be
	// immediately effective without manual permission reload.
	ctx := context.Background()
	p := testutil.NewPersistence(t)
	ac := iam.CreateAccessController(iam.AccessControllerOptions{},
		slog.New(slog.NewTextHandler(discarder{}, nil)))
	live := newRuleStore(t, p, ac)

	live.Set("public", func(req iam.AccessRequest) bool { return true })

	permMgr := NewMapPermissionManager()
	permMgr.RegisterScope("system:test:op", "auto_test", "")

	disp := NewSecureDispatcher(noopDispatcher{}, permMgr, ac)

	_, err := disp.Send(testMessage{ctx: anonymousContext(), name: "system:test:op"})
	if err == nil {
		t.Fatal("expected error for unregistered rule 'auto_test' before creation")
	}
	if !strings.Contains(err.Error(), "access denied") && !strings.Contains(err.Error(), "not registered") {
		t.Fatalf("expected access denial, got: %v", err)
	}

	doc := data.MustNewDocument(map[string]any{
		"name":       "auto_test",
		"ruleType":   "simple",
		"syntax":     "cel",
		"expression": "true",
	}, ctx)
	_, err = live.CreateOne(ctx, doc)
	if err != nil {
		t.Fatalf("CreateOne: %v", err)
	}

	_, err = disp.Send(testMessage{ctx: anonymousContext(), name: "system:test:op"})
	if err != nil {
		t.Fatalf("expected no error after rule created via LiveCollection, got: %v", err)
	}
}

func strPtr(s string) *string { return &s }
