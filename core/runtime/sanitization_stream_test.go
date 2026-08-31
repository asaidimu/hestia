package runtime

import (
	"context"
	"testing"

	"github.com/asaidimu/go-anansi/v8/core/common"
	"github.com/asaidimu/go-anansi/v8/core/document"

	"github.com/asaidimu/hestia/core/abstract"
)

type fakeMessage struct {
	abstract.Message
	name string
	ctx  context.Context
}

func (m *fakeMessage) Name() string             { return m.name }
func (m *fakeMessage) Context() context.Context { return m.ctx }

type captureDispatcher struct {
	completeWith *abstract.Result
}

func (d *captureDispatcher) Send(ctx context.Context, msg abstract.Message, onComplete abstract.CompletionFunc) error {
	if d.completeWith != nil {
		abstract.Complete(onComplete, ctx, d.completeWith, nil)
	}
	return nil
}

func (d *captureDispatcher) Wrap(next abstract.Dispatcher) abstract.Dispatcher { return d }

// TestSanitizationDispatcherStashesScopeForStreams pins the streamed-result
// contract: the dispatcher cannot wrap a DocumentChannel (writer-owned stream
// teardown per S-19), so it stashes the message's feature scope on the result
// metadata for the transport to apply per document.
func TestSanitizationDispatcherStashesScopeForStreams(t *testing.T) {
	ch := make(chan *document.Document, 1)
	close(ch)
	res := &abstract.Result{Kind: abstract.ResultKindDocumentChannel, DocumentChannel: ch}
	d := NewSanitizationDispatcher(&captureDispatcher{completeWith: res})

	if err := d.Send(context.Background(), &fakeMessage{name: "system:audit:log:stream", ctx: context.Background()}, func(context.Context, *abstract.Result, error) {}); err != nil {
		t.Fatalf("Send: %v", err)
	}

	scopes, ok := res.Metadata[ResultSanitizationScopeMetaKey].([]string)
	if !ok || len(scopes) != 1 || scopes[0] != "audit" {
		t.Fatalf("metadata scope = %v, want [audit]", res.Metadata[ResultSanitizationScopeMetaKey])
	}
}

// TestSanitizationDispatcherSkipsScopeForNonStreams pins that non-streamed
// results carry no stash key: their sanitization happens inline.
func TestSanitizationDispatcherSkipsScopeForNonStreams(t *testing.T) {
	res := &abstract.Result{Kind: abstract.ResultKindDocument}
	d := NewSanitizationDispatcher(&captureDispatcher{completeWith: res})

	if err := d.Send(context.Background(), &fakeMessage{name: "system:users:user:get", ctx: context.Background()}, func(_ context.Context, out *abstract.Result, err error) {
		if out.Metadata != nil {
			if _, present := out.Metadata[ResultSanitizationScopeMetaKey]; present {
				t.Error("scope stashed for a non-streamed result")
			}
		}
	}); err != nil {
		t.Fatalf("Send: %v", err)
	}
}

// TestStreamSanitizeArgsConsumesScope pins the transport side: the stashed
// scope re-materializes as a scoped context and the metadata key is consumed
// so it never leaks into response metadata.
func TestStreamSanitizeArgsConsumesScope(t *testing.T) {
	ch := make(chan *document.Document, 1)
	close(ch)
	res := &abstract.Result{
		Kind:            abstract.ResultKindDocumentChannel,
		DocumentChannel: ch,
		Metadata:        map[string]any{ResultSanitizationScopeMetaKey: []string{"audit"}},
	}

	args := StreamSanitizeArgs(context.Background(), res)
	if len(args) == 0 {
		t.Fatal("no sanitize contexts returned")
	}
	scopes := common.SanitizationScopesFromContext(args[0])
	if len(scopes) != 1 || scopes[0] != "audit" {
		t.Fatalf("scopes from ctx = %v, want [audit]", scopes)
	}
	if _, present := res.Metadata[ResultSanitizationScopeMetaKey]; present {
		t.Fatal("metadata key not consumed — it would leak into response metadata")
	}
	if StreamSanitizeArgs(context.Background(), res) != nil {
		t.Fatal("second call must find nothing")
	}
}
