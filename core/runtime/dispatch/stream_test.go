package dispatch

import (
        "context"
        "sync"
        "testing"
        "time"

        "github.com/asaidimu/go-anansi/v8/core/data"
        "github.com/asaidimu/go-anansi/v8/core/document"

        "github.com/asaidimu/hestia/core/abstract"
)

type streamTestItem struct {
        Name string `input:"payload.name"`
}

// chanMessage is a minimal abstract.Message over a fixed input channel.
type chanMessage struct {
        abstract.Message
        id      string
        name    string
        ctx     context.Context
        inputCh <-chan abstract.StreamItem
}

func (m *chanMessage) ID() string                             { return m.id }
func (m *chanMessage) Name() string                           { return m.name }
func (m *chanMessage) Context() context.Context               { return m.ctx }
func (m *chanMessage) Input() data.Documenter                 { return nil }
func (m *chanMessage) InputChannel() <-chan abstract.StreamItem { return m.inputCh }
func (m *chanMessage) BlobInputChannel() <-chan abstract.Blob { return nil }
func (m *chanMessage) TenantID() string                       { return "" }
func (m *chanMessage) TraceID() string                        { return "" }
func (m *chanMessage) RequestID() string                      { return "" }
func (m *chanMessage) SourceIP() string                       { return "" }
func (m *chanMessage) UserAgent() string                      { return "" }
func (m *chanMessage) ResourceID() string                     { return "" }
func (m *chanMessage) SessionID() string                      { return "" }

func makeItemDoc(t *testing.T, pool *document.DocumentPool, payload string) *document.Document {
        t.Helper()
        doc, err := pool.FromJSON([]byte(`{"payload":` + payload + `}`))
        if err != nil {
                t.Fatalf("FromJSON: %v", err)
        }
        return doc
}

// TestHandleInputStreamBindsValuesAndSurfacesErrors pins the adapter contract:
// valid items bind into TIn via the input tag; StreamItem errors surface as
// Item.Err with the handler free to continue; every document is released
// exactly once (idempotent Release + -race cover the double-release hazard).
func TestHandleInputStreamBindsValuesAndSurfacesErrors(t *testing.T) {
        schema := SchemaFromTypeWithTag[streamTestItem]("input")
        pool, err := document.NewDocumentPool(schema)
        if err != nil {
                t.Fatalf("pool: %v", err)
        }

        ch := make(chan abstract.StreamItem, 3)
        ch <- abstract.StreamItem{Doc: makeItemDoc(t, pool, `{"name":"a"}`)}
        ch <- abstract.StreamItem{Err: context.DeadlineExceeded}
        ch <- abstract.StreamItem{Doc: makeItemDoc(t, pool, `{"name":"b"}`)}
        close(ch)

        msg := &chanMessage{id: MustNewID(), name: "test:stream", ctx: context.Background(), inputCh: ch}

        var mu sync.Mutex
        var values []string
        var errs []error
        handler := HandleInputStream[streamTestItem](func(ctx context.Context, msg abstract.Message, items <-chan Item[streamTestItem]) (*abstract.Result, error) {
                for it := range items {
                        mu.Lock()
                        if it.Err != nil {
                                errs = append(errs, it.Err)
                        } else {
                                values = append(values, it.Value.Name)
                        }
                        mu.Unlock()
                }
                return &abstract.Result{}, nil
        })

        done := make(chan struct{})
        go func() {
                _, _ = handler(context.Background(), msg)
                close(done)
        }()

        select {
        case <-done:
        case <-time.After(2 * time.Second):
                t.Fatal("handler deadlocked on input stream")
        }

        mu.Lock()
        defer mu.Unlock()
        if len(values) != 2 || values[0] != "a" || values[1] != "b" {
                t.Fatalf("values = %v, want [a b]", values)
        }
        if len(errs) != 1 {
                t.Fatalf("errs = %v, want exactly the one item error", errs)
        }
}

// TestHandleInputStreamAbandonedStreamDrainsProducer pins the lifecycle
// guarantee: when the handler returns early (before the stream ends), the
// adapter keeps draining the producer to close so the body reader can run to
// EOF instead of blocking forever, and the handler call returns promptly.
func TestHandleInputStreamAbandonedStreamDrainsProducer(t *testing.T) {
        schema := SchemaFromTypeWithTag[streamTestItem]("input")
        pool, err := document.NewDocumentPool(schema)
        if err != nil {
                t.Fatalf("pool: %v", err)
        }

        ch := make(chan abstract.StreamItem)
        msg := &chanMessage{id: MustNewID(), name: "test:stream", ctx: context.Background(), inputCh: ch}

        handler := HandleInputStream[streamTestItem](func(ctx context.Context, msg abstract.Message, items <-chan Item[streamTestItem]) (*abstract.Result, error) {
                <-items // consume exactly one item, then abandon the stream
                return &abstract.Result{}, nil
        })

        // Producer: keeps feeding after the handler abandoned the stream.
        producedOK := make(chan bool, 1)
        go func() {
                defer close(ch)
                for i := 0; i < 100; i++ {
                        doc := makeItemDoc(t, pool, `{"name":"x"}`)
                        select {
                        case ch <- abstract.StreamItem{Doc: doc}:
                        case <-time.After(2 * time.Second):
                                producedOK <- false
                                return
                        }
                }
                producedOK <- true
        }()

        done := make(chan struct{})
        go func() {
                _, _ = handler(context.Background(), msg)
                close(done)
        }()

        select {
        case <-done:
        case <-time.After(2 * time.Second):
                t.Fatal("handler call blocked — abandoned stream was not drained")
        }
        select {
        case ok := <-producedOK:
                if !ok {
                        t.Fatal("producer blocked feeding an abandoned stream (drain-and-release not working)")
                }
        case <-time.After(2 * time.Second):
                t.Fatal("producer never finished")
        }
}

// TestDispatchDocumentStreamForwardsChannel pins the dispatch contract: with
// DispatchInput.DocumentStream set, the message's InputChannel is exactly the
// producer's channel and no readiness barrier exists (Input stays nil, no
// post-completion signal is sent).
func TestDispatchDocumentStreamForwardsChannel(t *testing.T) {
        docCh := make(chan abstract.StreamItem, 1)
        docCh <- abstract.StreamItem{Err: errMarker{}}

        var sawItem bool
        var sawChannel bool
        var singleInput data.Documenter
        disp := &inlineDispatcher{handler: func(ctx context.Context, msg abstract.Message) (*abstract.Result, error) {
                sawChannel = msg.InputChannel() != nil
                singleInput = msg.Input()
                si, ok := <-msg.InputChannel()
                sawItem = ok && si.Err == errMarker{}
                return &abstract.Result{}, nil
        }}

        if _, err := Dispatch(context.Background(), disp, DispatchInput{
                Name:           "test:stream:import",
                Context:        context.Background(),
                DocumentStream: docCh,
                Intent:         abstract.Create,
        }); err != nil {
                t.Fatalf("Dispatch: %v", err)
        }

        if !sawChannel {
                t.Fatal("handler saw a nil InputChannel")
        }
        if singleInput != nil {
                t.Fatal("streaming dispatch must not carry a single input document")
        }
        if !sawItem {
                t.Fatal("the producer's item did not flow through the handler's InputChannel")
        }
}

type errMarker struct{}

func (errMarker) Error() string { return "producer item" }

// inlineDispatcher runs the registered handler synchronously inside Send —
// the minimum abstract.Dispatcher the forwarding test needs.
type inlineDispatcher struct {
        handler abstract.MessageHandler
}

func (d *inlineDispatcher) Send(ctx context.Context, msg abstract.Message, onComplete abstract.CompletionFunc) error {
        res, err := d.handler(ctx, msg)
        abstract.Complete(onComplete, ctx, res, err)
        return nil
}

func (d *inlineDispatcher) Wrap(next abstract.Dispatcher) abstract.Dispatcher { return d }
