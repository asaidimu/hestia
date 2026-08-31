package http

import (
        "context"
        "strings"
        "sync"
        "testing"
        "time"

        "github.com/asaidimu/hestia/core/abstract"
        dispatch "github.com/asaidimu/hestia/core/runtime/dispatch"
        "github.com/asaidimu/hestia/core/runtime"
)

// streamingHarness installs a streaming registration backed by a real
// LocalDispatcher and returns the installed route handler.
type streamResult struct {
        mu     sync.Mutex
        values []string
        errs   []error
}

func streamingMock(t *testing.T, reg abstract.MessageRegistration, collect *streamResult) Handler {
        t.Helper()
        disp := runtime.NewLocalDispatcherWithLogger(nil)
        handler := dispatch.HandleInputStream[importItem](func(ctx context.Context, msg abstract.Message, items <-chan dispatch.Item[importItem]) (*abstract.Result, error) {
                for it := range items {
                        if it.Err != nil {
                                collect.mu.Lock()
                                collect.errs = append(collect.errs, it.Err)
                                collect.mu.Unlock()
                                continue
                        }
                        collect.mu.Lock()
                        collect.values = append(collect.values, it.Value.Name)
                        collect.mu.Unlock()
                }
                return &abstract.Result{}, nil
        })
        if err := disp.RegisterHandler(reg.Name, handler, abstract.HandlerInfo{Name: reg.Name, Enabled: true}); err != nil {
                t.Fatalf("RegisterHandler: %v", err)
        }

        mt := newMockTransport()
        orch := &Interface{
                trans:        mt,
                disp:         disp,
                regs:         []abstract.MessageRegistration{reg},
                bootstrapped: true,
        }
        orch.installDispatcherRegistrations()

        mt.mu.Lock()
        defer mt.mu.Unlock()
        h, ok := mt.handlers["POST /system/users/user/import"]
        if !ok {
                t.Fatal("streaming route not installed")
        }
        return h
}

type importItem struct {
        // Each NDJSON item plays the role of the request body, so item fields
        // bind through the payload section exactly like a buffered body.
        Name string `input:"payload.name" anansi:"required=true"`
}

func streamingRegistration(t *testing.T) abstract.MessageRegistration {
        t.Helper()
        return abstract.MessageRegistration{
                Name:   "system:users:user:import",
                Intent: abstract.Create,
                Input: abstract.Input{
                        Schema:    dispatch.SchemaFromTypeWithTag[importItem]("input"),
                        Streaming: true,
                },
                Enabled: true,
        }
}

// TestStreamingRouteEndToEnd pins the whole input-streaming path: a streamed
// body is framed as NDJSON, each item is enveloped and validated by the
// producer, bound into TIn by HandleInputStream, and the handler's summary is
// the HTTP response.
func TestStreamingRouteEndToEnd(t *testing.T) {
        collect := &streamResult{}
        h := streamingMock(t, streamingRegistration(t), collect)

        body := "{\"name\":\"alice\"}\n{\"name\":\"bob\"}\n"
        resp, err := h(context.Background(), Request{
                BodyStream: strings.NewReader(body),
                Headers:    map[string][]string{},
        })
        if err != nil {
                t.Fatalf("handler error: %v", err)
        }
        if resp.Status != statusOK && resp.Status != statusCreated {
                t.Fatalf("status = %d", resp.Status)
        }

        deadline := time.After(time.Second)
        for {
                collect.mu.Lock()
                done := len(collect.values) == 2
                collect.mu.Unlock()
                if done {
                        break
                }
                select {
                case <-deadline:
                        t.Fatalf("handler never consumed both items; got %v", collect.values)
                default:
                        time.Sleep(time.Millisecond)
                }
        }
        if collect.values[0] != "alice" || collect.values[1] != "bob" {
                t.Fatalf("values = %v, want [alice bob]", collect.values)
        }
        if len(collect.errs) != 0 {
                t.Fatalf("unexpected errors: %v", collect.errs)
        }
}

// TestStreamingRouteFramingErrorEndsStream pins the framing contract: a
// non-EOF decode failure emits one per-item error and ends the stream — the
// handler sees the error, the request still succeeds (the handler owns the
// abort-vs-collect policy), and items after the corruption never arrive.
func TestStreamingRouteFramingErrorEndsStream(t *testing.T) {
        collect := &streamResult{}
        h := streamingMock(t, streamingRegistration(t), collect)

        body := "{\"name\":\"alice\"}\n{not json\n{\"name\":\"never\"}\n"
        if _, err := h(context.Background(), Request{BodyStream: strings.NewReader(body), Headers: map[string][]string{}}); err != nil {
                t.Fatalf("handler error: %v", err)
        }

        deadline := time.After(time.Second)
        for {
                collect.mu.Lock()
                done := len(collect.errs) > 0
                collect.mu.Unlock()
                if done {
                        break
                }
                select {
                case <-deadline:
                        t.Fatal("framing error never surfaced as an item error")
                default:
                        time.Sleep(time.Millisecond)
                }
        }
        collect.mu.Lock()
        defer collect.mu.Unlock()
        if len(collect.values) != 1 || collect.values[0] != "alice" {
                t.Fatalf("values = %v, want [alice] — items after framing corruption must not arrive", collect.values)
        }
}

// TestStreamingRouteValidationFailureContinues pins the per-item validation
// policy: an item failing schema validation surfaces as Item.Err and the
// stream continues — the handler decides what a failed item means.
func TestStreamingRouteValidationFailureContinues(t *testing.T) {
        collect := &streamResult{}
        reg := streamingRegistration(t)
        h := streamingMock(t, reg, collect)

        // "name" is required by the generated schema (anansi tag on the field);
        // the second item omits it.
        body := "{\"name\":\"alice\"}\n{}\n{\"name\":\"carol\"}\n"
        if _, err := h(context.Background(), Request{BodyStream: strings.NewReader(body), Headers: map[string][]string{}}); err != nil {
                t.Fatalf("handler error: %v", err)
        }

        deadline := time.After(time.Second)
        for {
                collect.mu.Lock()
                done := len(collect.errs) > 0 && len(collect.values) == 2
                collect.mu.Unlock()
                if done {
                        break
                }
                select {
                case <-deadline:
                        collect.mu.Lock()
                        defer collect.mu.Unlock()
                        t.Fatalf("want 2 values + 1 error, got values=%v errs=%v", collect.values, collect.errs)
                default:
                        time.Sleep(time.Millisecond)
                }
        }
        collect.mu.Lock()
        defer collect.mu.Unlock()
        if collect.values[0] != "alice" || collect.values[1] != "carol" {
                t.Fatalf("values = %v, want [alice carol]", collect.values)
        }
}

// TestStreamingRouteRequiresBodyStream pins the guard: a streaming route hit
// without a streamed body fails loudly instead of dispatching an empty stream.
func TestStreamingRouteRequiresBodyStream(t *testing.T) {
        collect := &streamResult{}
        h := streamingMock(t, streamingRegistration(t), collect)

        _, err := h(context.Background(), Request{Headers: map[string][]string{}})
        if err == nil {
                t.Fatal("expected error for missing body stream")
        }
}
