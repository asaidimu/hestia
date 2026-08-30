// @note #arch-20260821-004 issue resolved priority=P2 tags=#arch,#duplication : Separate rate store instances per dispatcher
// @assignee opencode
// @see #arch-20260821-003
// Shared single ratestore.InMemoryStore between RateLimitDispatcher and ThrottleDispatcher via ProviderSet.RateStore. Created once in DispatcherChain, closed on Stop().
//
// RateLimitDispatcher creates its own ratestore.New() instance at line 61.
// ThrottleDispatcher (throttle.go:78) creates a separate ratestore.New() instance.
//
// Each dispatcher creates an independent InMemoryStore instance with its own
// eviction goroutine and shard array. This doubles memory usage and means
// rate limits and throttles are tracked in separate stores even when they
// could share infrastructure.
//
// Resolution: Pass the RateLimitStore via DI or constructor instead of creating
// new instances internally in each dispatcher. This allows sharing a single
// store instance across dispatchers.
package runtime

import (
	"context"
	"strings"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/runtime/ratestore"
)

// RateLimitError carries rate-limit state for the transport layer to set
// headers on 429 responses.
type RateLimitError struct {
	Remaining int
	ResetAt   int64
}

func (e *RateLimitError) Error() string { return "rate limit exceeded" }
func (e *RateLimitError) Unwrap() error { return ErrRateLimited }
func (e *RateLimitError) RetryAfter() int {
	d := time.Until(time.Unix(e.ResetAt, 0))
	if d < 0 {
		return 0
	}
	return int(d.Seconds())
}

type RateLimitStore interface {
	CheckAndConsume(ctx context.Context, key string, burst int64, tokensPerPeriod int64, period time.Duration) (int, bool, error)
	Increment(ctx context.Context, key string, window time.Duration) (int, error)
	Reset(ctx context.Context, key string) error
}

type MatchIdentity string

const (
	MatchIdentityUser   MatchIdentity = "user"
	MatchIdentityIP     MatchIdentity = "ip"
	MatchIdentityAPIKey MatchIdentity = "apikey"
	MatchIdentityTenant MatchIdentity = "tenant"
	MatchIdentityGlobal MatchIdentity = "global"
)

type RateLimitLookup func(operation string) *RateLimitPolicy

type RateLimitDispatcher struct {
	next     abstract.Dispatcher
	lookup   RateLimitLookup
	store    RateLimitStore
	logger   *zap.Logger
	storeErrs atomic.Int64
}

func NewRateLimitDispatcher(lookup RateLimitLookup, store RateLimitStore, logger ...*zap.Logger) *RateLimitDispatcher {
	if lookup == nil {
		lookup = func(string) *RateLimitPolicy { return nil }
	}
	if store == nil {
		store = ratestore.New()
	}
	var log *zap.Logger
	if len(logger) > 0 && logger[0] != nil {
		log = logger[0]
	} else {
		log = zap.NewNop()
	}
	return &RateLimitDispatcher{
		lookup: lookup,
		store:  store,
		logger: log,
	}
}

func (d *RateLimitDispatcher) Wrap(next abstract.Dispatcher) abstract.Dispatcher {
	return &RateLimitDispatcher{
		next:   next,
		lookup: d.lookup,
		store:  d.store,
		logger: d.logger,
	}
}

func (d *RateLimitDispatcher) Send(ctx context.Context, msg abstract.Message, onComplete abstract.CompletionFunc) error {
	rate := d.lookup(msg.Name())
	if rate == nil {
		return d.next.Send(ctx, msg, onComplete)
	}

	key := buildRateLimitKey(rate.Identity, msg)
	remaining, ok, err := d.store.CheckAndConsume(msg.Context(), key, rate.Capacity, rate.Refill, time.Duration(rate.Period)*time.Second)
	if err != nil {
		// S-21: silent fail-open meant an attacker who can induce store
		// errors disables rate limiting with no signal. The request still
		// proceeds (fail-closed here would turn a store outage into a
		// full outage), but consecutive failures are counted and logged
		// loudly so operators can alert on them.
		n := d.storeErrs.Add(1)
		d.logger.Error("rate limit store error; failing open",
			zap.Int64("consecutive_failures", n), zap.Error(err))
		return d.next.Send(ctx, msg, onComplete)
	}
	d.storeErrs.Store(0)
	if !ok {
		return &RateLimitError{
			Remaining: 0,
			ResetAt:   time.Now().Add(time.Duration(rate.Period) * time.Second).Unix(),
		}
	}

	return d.next.Send(ctx, msg, func(cctx context.Context, res *abstract.Result, handlerErr error) {
		if handlerErr == nil && res != nil {
			if res.Metadata == nil {
				res.Metadata = make(map[string]any)
			}
			res.Metadata[MetaKeyRates] = &RateLimitMeta{
				Remaining: remaining,
				Limit:     int(rate.Capacity),
				ResetAt:   time.Now().Add(time.Duration(rate.Period) * time.Second).Unix(),
			}
		}
		onComplete(cctx, res, handlerErr)
	})
}

func buildRateLimitKey(identity string, msg abstract.Message) string {
	op := msg.Name()
	var dimension string
	switch MatchIdentity(identity) {
	case MatchIdentityUser:
		dimension = extractUserID(msg.Context())
	case MatchIdentityIP:
		dimension = msg.SourceIP()
	case MatchIdentityAPIKey:
		dimension = extractAPIKeyID(msg.Context())
	case MatchIdentityTenant:
		dimension = msg.TenantID()
	default:
		dimension = ""
	}
	if dimension == "" {
		return "rate:global:" + op
	}
	return "rate:" + identity + ":" + op + ":" + dimension
}

func extractUserID(ctx context.Context) string {
	return GetUserID(ctx)
}

func extractAPIKeyID(ctx context.Context) string {
	return GetTokenID(ctx)
}

func matchOperation(pattern, operation string) bool {
	if pattern == "" || pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(operation, prefix)
	}
	return pattern == operation
}

var _ abstract.Dispatcher = (*RateLimitDispatcher)(nil)
var _ abstract.DispatcherLink = (*RateLimitDispatcher)(nil)
