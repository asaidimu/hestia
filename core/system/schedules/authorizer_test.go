package schedules_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/collection"
	"github.com/asaidimu/go-iam/v2/iam"
	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	runtime "github.com/asaidimu/hestia/core/runtime"
	runtimecontext "github.com/asaidimu/hestia/core/runtime/context"
	dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
	"github.com/asaidimu/hestia/core/runtime/scheduler"
	schedules "github.com/asaidimu/hestia/core/system/schedules"
	"github.com/asaidimu/hestia/core/system/schedules/model"
	"github.com/asaidimu/hestia/core/system/policies"
	"github.com/asaidimu/hestia/core/system/users"
)

// fakePermMgr resolves target operations to fixed rule keys.
type fakePermMgr struct {
	rules map[string]string
}

func (f *fakePermMgr) Resolve(msg abstract.Message) (string, bool, error) {
	if f.rules == nil {
		return "", false, errorNoPolicy
	}
	rule, ok := f.rules[msg.Name()]
	if !ok {
		return "", false, errorNoPolicy
	}
	return rule, true, nil
}

func (f *fakePermMgr) ListCapabilities() []runtime.CapabilityMetadata { return nil }

var errorNoPolicy = errors.New("no policy binding")

// fakeLiveUsers stubs the user claims cache. Only Get is consulted by the
// authorizer; the embedded nil interface covers the rest of LiveCollection.
type fakeLiveUsers struct {
	collection.LiveCollection[*users.UserClaims]
	users map[string]*users.UserClaims
}

func (f *fakeLiveUsers) Get(key string) (*users.UserClaims, bool) {
	u, ok := f.users[key]
	return u, ok
}

func testAccessCtrl() iam.AccessController {
	return iam.CreateAccessController(iam.AccessControllerOptions{
		Rules: policies.GoDefaultRules(),
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func callerCtx(userID string, scopes []string) context.Context {
	return runtimecontext.ContextWithClaims(context.Background(), &abstract.Claims{
		UserID: userID,
		Scopes: scopes,
	})
}

// TestAuthorizeTargetDeniesPrivilegedTarget: an authenticated caller cannot
// schedule an operation bound to a rule they do not satisfy (here
// "administrator").
func TestAuthorizeTargetDeniesPrivilegedTarget(t *testing.T) {
	auth := schedules.NewScheduleAuthorizer(
		&fakePermMgr{rules: map[string]string{"system:users:user:create": "administrator"}},
		testAccessCtrl(),
		nil,
	)
	if err := auth.AuthorizeTarget(callerCtx("user-1", nil), "system:users:user:create"); err == nil {
		t.Fatal("expected denial scheduling an administrator-only operation as a regular user")
	}
}

// TestAuthorizeTargetAllowsMatchingRule: a caller who could invoke the target
// directly may schedule it.
func TestAuthorizeTargetAllowsMatchingRule(t *testing.T) {
	auth := schedules.NewScheduleAuthorizer(
		&fakePermMgr{rules: map[string]string{
			"system:notifications:notification:create": "authenticated",
			"system:settings:set":                      "administrator",
		}},
		testAccessCtrl(),
		nil,
	)
	if err := auth.AuthorizeTarget(callerCtx("user-1", nil), "system:notifications:notification:create"); err != nil {
		t.Fatalf("expected authenticated-rule target to be schedulable: %v", err)
	}
	if err := auth.AuthorizeTarget(callerCtx("admin-1", []string{"administrator"}), "system:settings:set"); err != nil {
		t.Fatalf("expected administrator to be able to schedule administrator-rule target: %v", err)
	}
	if err := auth.AuthorizeTarget(callerCtx("user-1", nil), "system:settings:set"); err == nil {
		t.Fatal("expected non-administrator to be denied scheduling an administrator-rule target")
	}
}

// TestAuthorizeTargetFailsClosed: unresolvable policy bindings deny.
func TestAuthorizeTargetFailsClosed(t *testing.T) {
	auth := schedules.NewScheduleAuthorizer(&fakePermMgr{}, testAccessCtrl(), nil)
	if err := auth.AuthorizeTarget(callerCtx("user-1", nil), "system:unknown:op"); err == nil {
		t.Fatal("expected denial for target without resolvable policy binding")
	}
}

// TestFireClaimsUsesCurrentScopes: fire-time claims re-read the creator's
// CURRENT permissions and tenant from the claims cache.
func TestFireClaimsUsesCurrentScopes(t *testing.T) {
	cache := &fakeLiveUsers{users: map[string]*users.UserClaims{
		"user-1": {UserID: "user-1", Email: "u1@example.com", TenantID: "tenant-9", Permissions: []string{"administrator"}},
	}}
	auth := schedules.NewScheduleAuthorizer(nil, nil, func() collection.LiveCollection[*users.UserClaims] { return cache })

	claims := auth.FireClaims("user-1", "tenant-stale")
	if claims == nil {
		t.Fatal("expected claims for active creator")
	}
	if claims.UserID != "user-1" || claims.Email != "u1@example.com" || claims.TenantID != "tenant-9" {
		t.Errorf("claims = %+v, want user-1/u1@example.com/tenant-9", claims)
	}
	if len(claims.Scopes) != 1 || claims.Scopes[0] != "administrator" {
		t.Errorf("scopes = %v, want [administrator]", claims.Scopes)
	}
}

// TestFireClaimsFailsClosed: creators that no longer resolve to an active
// account (deleted, disabled) produce nil claims — the fire must be skipped.
func TestFireClaimsFailsClosed(t *testing.T) {
	cache := &fakeLiveUsers{users: map[string]*users.UserClaims{}}
	auth := schedules.NewScheduleAuthorizer(nil, nil, func() collection.LiveCollection[*users.UserClaims] { return cache })
	if claims := auth.FireClaims("ghost", ""); claims != nil {
		t.Fatalf("expected nil claims for missing creator, got %+v", claims)
	}
	if claims := auth.FireClaims("", ""); claims != nil {
		t.Fatalf("expected nil claims for empty creator, got %+v", claims)
	}
}

// TestFireClaimsFallbackWithoutCache: without a claims cache (unit tests),
// fires degrade to a scope-less creator identity instead of failing.
func TestFireClaimsFallbackWithoutCache(t *testing.T) {
	auth := schedules.NewScheduleAuthorizer(nil, nil, nil)
	claims := auth.FireClaims("user-1", "tenant-1")
	if claims == nil || claims.UserID != "user-1" || claims.TenantID != "tenant-1" || len(claims.Scopes) != 0 {
		t.Fatalf("fallback claims = %+v, want scope-less creator identity", claims)
	}
}

// TestCreateScheduleRejectsForeignAttribution: a caller may not create a
// schedule attributed to another user.
func TestCreateScheduleRejectsForeignAttribution(t *testing.T) {
	m := newTestModel(t)
	log := zap.NewNop()
	sched := scheduler.New(context.Background(), log)
	disp := runtime.NewLocalDispatcher()
	live := schedules.NewLiveSchedule(m, sched, disp, log)

	svc := schedules.NewSchedulesServiceForTest(m, live)
	_, err := svc.Create(callerCtx("user-1", nil), &testMessage{}, &model.ScheduleCreateInput{
		Message: "system:test:handler",
		Cron:    "@every 1h",
		UserID:  "user-2",
	})
	if err == nil {
		t.Fatal("expected denial creating a schedule attributed to another user")
	}
}

// TestLiveScheduleSkipsFireWhenCreatorInactive: end to end — a schedule whose
// creator no longer resolves to an active account must NOT fire, while a
// schedule with an active creator still does.
func TestLiveScheduleSkipsFireWhenCreatorInactive(t *testing.T) {
	ctx := context.Background()
	log := zap.NewNop()

	m := newTestModel(t)
	disp := runtime.NewLocalDispatcher()

	var callCount atomic.Int64
	var ghostCount atomic.Int64
	err := disp.RegisterHandler("system:notifications:notification:create",
		dispatch.Handle(func(ctx context.Context, msg abstract.Message, input *notifCreateDTO) (*abstract.Result, error) {
			if input.UserID == "ghost" {
				ghostCount.Add(1)
			}
			callCount.Add(1)
			return &abstract.Result{}, nil
		}),
		abstract.HandlerInfo{Description: "test notification create", Enabled: true},
	)
	if err != nil {
		t.Fatalf("register handler: %v", err)
	}

	cache := &fakeLiveUsers{users: map[string]*users.UserClaims{
		"active": {UserID: "active", Permissions: []string{}},
	}}
	auth := schedules.NewScheduleAuthorizer(nil, nil, func() collection.LiveCollection[*users.UserClaims] { return cache })

	schedCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sched := scheduler.New(schedCtx, log)
	live := schedules.NewLiveSchedule(m, sched, disp, log).
		WithAuthorizer(auth)
	for _, creator := range []string{"ghost", "active"} {
		doc := data.MustNewDocument(map[string]any{
			"user_id": creator,
			"message": "system:notifications:notification:create",
			"input": map[string]any{
				"subject": "tick",
				"body":    "body for " + creator,
				"type":    "reminder",
				"user_id": creator,
			},
			"cron": "@every 1s",
		})
		if _, err := m.CreateSchedule(ctx, doc); err != nil {
			t.Fatalf("create schedule for %s: %v", creator, err)
		}
	}

	if err := live.Init(ctx); err != nil {
		t.Fatalf("LiveSchedule.Init: %v", err)
	}

	sched.Start()
	defer sched.Stop()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if callCount.Load() > 0 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if callCount.Load() == 0 {
		t.Fatal("active creator's schedule never fired — authorizer broke legitimate fires")
	}
	if g := ghostCount.Load(); g != 0 {
		t.Fatalf("inactive creator's schedule fired %d time(s) — fail-closed violated", g)
	}
}
