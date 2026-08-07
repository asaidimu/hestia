package runtime

import (
	"context"
	"strings"
	"time"

	"github.com/asaidimu/go-iam/v2/iam"

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

func NewRateLimitDispatcher(lookup RateLimitLookup) *RateLimitDispatcher {
	if lookup == nil {
		lookup = func(string) *RateLimitPolicy { return nil }
	}
	return &RateLimitDispatcher{
		lookup: lookup,
		store:  ratestore.New(),
	}
}

func (d *RateLimitDispatcher) Wrap(next abstract.Dispatcher) abstract.Dispatcher {
	return &RateLimitDispatcher{
		next:   next,
		lookup: d.lookup,
		store:  d.store,
	}
}

func (d *RateLimitDispatcher) Send(msg abstract.Message) (*abstract.Result, error) {
	rate := d.lookup(msg.Name())
	if rate == nil {
		return d.next.Send(msg)
	}

	key := buildRateLimitKey(rate.Identity, msg)
	remaining, ok, err := d.store.CheckAndConsume(msg.Context(), key, rate.Capacity, rate.Refill, time.Duration(rate.Period)*time.Second)
	if err != nil {
		return d.next.Send(msg)
	}
	if !ok {
		return nil, &RateLimitError{
			Remaining: 0,
			ResetAt:   time.Now().Add(time.Duration(rate.Period) * time.Second).Unix(),
		}
	}

	result, err := d.next.Send(msg)
	if err != nil {
		return nil, err
	}
	if result.Metadata == nil {
		result.Metadata = make(map[string]any)
	}
	result.Metadata[MetaKeyRates] = &RateLimitMeta{
		Remaining: remaining,
		Limit:     int(rate.Capacity),
		ResetAt:   time.Now().Add(time.Duration(rate.Period) * time.Second).Unix(),
	}
	return result, nil
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
	ident, ok := iam.GetIdentity(ctx)
	if !ok {
		return ""
	}
	props, ok := ident.Properties.(map[string]any)
	if !ok {
		return ""
	}
	uid, _ := props["user_id"].(string)
	return uid
}

func extractAPIKeyID(ctx context.Context) string {
	ident, ok := iam.GetIdentity(ctx)
	if !ok {
		return ""
	}
	props, ok := ident.Properties.(map[string]any)
	if !ok {
		return ""
	}
	tid, _ := props["token_id"].(string)
	return tid
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
