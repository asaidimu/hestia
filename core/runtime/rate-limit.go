// @note #arch-20260821-004 issue resolved status=open priority=P2 tags=#arch,#duplication : Separate rate store instances per dispatcher
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
	"time"

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
	next   abstract.Dispatcher
	lookup RateLimitLookup
	store  RateLimitStore
}

func NewRateLimitDispatcher(lookup RateLimitLookup, store RateLimitStore) *RateLimitDispatcher {
	if lookup == nil {
		lookup = func(string) *RateLimitPolicy { return nil }
	}
	if store == nil {
		store = ratestore.New()
	}
	return &RateLimitDispatcher{
		lookup: lookup,
		store:  store,
	}
}

func (d *RateLimitDispatcher) Wrap(next abstract.Dispatcher) abstract.Dispatcher {
	return &RateLimitDispatcher{
		next:   next,
		lookup: d.lookup,
		store:  d.store,
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
		return d.next.Send(ctx, msg, onComplete)
	}
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
