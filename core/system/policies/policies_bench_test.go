package policies_test

import (
	"context"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/data"
	"github.com/asaidimu/go-anansi/v8/core/persistence/collection"

	"github.com/asaidimu/hestia/core/abstract"
	"github.com/asaidimu/hestia/core/internal/testutil"
	"github.com/asaidimu/hestia/core/system/policies"
)

// newBenchLiveManager builds a LivePermissionManager backed by a real
// LiveRepository over the _operation_policy_ collection, with one cached
// policy — the production shape.
func newBenchLiveManager(b *testing.B) *policies.LivePermissionManager {
	b.Helper()
	ctx := context.Background()
	p := testutil.NewPersistenceTB(b)
	opColl, err := p.Collection(ctx, "_operation_policy_")
	if err != nil {
		b.Fatalf("open _operation_policy_ collection: %v", err)
	}

	live, err := collection.NewLiveRepository(ctx, collection.LiveRepositoryOptions[*policies.Policy]{
		Collection: opColl,
		Processor:  &policies.PolicyDocProcessor{},
		QueryKey:   "key",
		AutoLoad:   false,
	})
	if err != nil {
		b.Fatalf("NewLiveRepository: %v", err)
	}
	b.Cleanup(func() { _ = live.Close() })

	live.Set(":bench:svc:op:run", &policies.Policy{
		Operation: "bench:svc:op:run",
		Rule:      "administrator",
		Key:           ":bench:svc:op:run",
		Enabled:       true,
	})

	return policies.NewLivePermissionManager(live, nil)
}

// BenchmarkLivePermissionManager_Resolve measures policy lookup on every
// dispatch (rule key resolution for authorization), including the tenant-key
// and global-key probe.
func BenchmarkLivePermissionManager_Resolve(b *testing.B) {
	m := newBenchLiveManager(b)
	msg := benchMessage{name: "bench:svc:op:run", ctx: context.Background()}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := m.Resolve(msg); err != nil {
			b.Fatal(err)
		}
	}
}

type benchMessage struct {
	name string
	ctx  context.Context
}

func (m benchMessage) ID() string                             { return "" }
func (m benchMessage) Name() string                           { return m.name }
func (m benchMessage) Context() context.Context               { return m.ctx }
func (m benchMessage) Input() data.Documenter                  { return nil }
func (m benchMessage) InputChannel() <-chan data.Documenter    { return nil }
func (m benchMessage) BlobInputChannel() <-chan abstract.Blob { return nil }
func (m benchMessage) TenantID() string                       { return "" }
func (m benchMessage) TraceID() string                        { return "" }
func (m benchMessage) RequestID() string                      { return "" }
func (m benchMessage) SourceIP() string                       { return "" }
func (m benchMessage) UserAgent() string                      { return "" }
func (m benchMessage) ResourceID() string                     { return "" }
func (m benchMessage) SessionID() string                      { return "" }

var _ abstract.Message = benchMessage{}