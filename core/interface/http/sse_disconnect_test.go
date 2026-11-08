package http

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
)

// TestSSEProducerReleasedOnClientDisconnect pins the S-19 contract: when an
// SSE client disconnects mid-stream, the producer goroutine (and whatever
// upstream feeds it) must be released. This is a runtime verification of the
// audit finding that fasthttp's RequestCtx.Done() returns the server
// shutdown channel (server.go: ctx.s.done) and NEVER fires on client
// disconnect — so the release has to be driven by the writer-owned done
// channel on managedStream.
//
// Layout:
//  1. streamChannel wraps a never-closing source channel (the "upstream").
//  2. A real fasthttp server writes the stream response via writeSuccess.
//  3. A raw TCP client reads the status line, then disconnects.
//  4. The writer's flush eventually fails → managedStream.Close() → the
//     producer must exit. We prove the producer exited by sending on the
//     unbuffered source channel: a send with no receiver blocks until the
//     probe timeout, which means nobody is draining upstream anymore.
func TestSSEProducerReleasedOnClientDisconnect(t *testing.T) {
	tr := &HTTPTransport{}

	src := make(chan any) // never closed, unbuffered
	resp, ok := streamChannel(context.Background(), src, func(v any) any { return v })
	if !ok {
		t.Fatal("streamChannel returned false")
	}
	ms, ok := resp.Body.(*managedStream)
	if !ok {
		t.Fatalf("Body = %T, want *managedStream", resp.Body)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	srv := &fasthttp.Server{Handler: func(ctx *fasthttp.RequestCtx) {
		tr.writeSuccess(ctx, resp)
	}}
	go func() {
		_ = srv.Serve(ln)
	}()
	defer func() {
		_ = srv.Shutdown()
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if _, err := conn.Write([]byte("GET / HTTP/1.1\r\nHost: test\r\n\r\n")); err != nil {
		t.Fatalf("write request: %v", err)
	}

	reader := bufio.NewReader(conn)
	if _, err := reader.ReadString('\n'); err != nil {
		t.Fatalf("read status line: %v", err)
	}
	_ = conn.Close() // client disconnects mid-stream

	// Feed events while the connection is dead. Each send must be consumed
	// by the producer; when the writer's flush finally fails against the
	// closed socket it closes ms.done, which stops both loops below.
	fed := 0
	deadline := time.Now().Add(10 * time.Second)
	for {
		select {
		case <-ms.done:
			goto released
		case src <- fmt.Sprintf("event-%d", fed):
			fed++
		case <-time.After(500 * time.Millisecond):
		}
		if time.Now().After(deadline) {
			t.Fatal("writer never closed the done channel after client disconnect")
		}
	}

released:
	// The producer must exit after the writer closes the done channel. Wait
	// for its explicit finished signal rather than racing a send against the
	// goroutine's select.
	select {
	case <-ms.finished:
	case <-time.After(2 * time.Second):
		t.Fatal("producer still running after client disconnect — goroutine leak (S-19)")
	}
	// And with the producer gone, an unbuffered send must find no receiver.
	select {
	case src <- "probe":
		t.Fatal("producer still consuming after client disconnect — goroutine leak (S-19)")
	case <-time.After(2 * time.Second):
	}
}

// TestSSEEventsReachClient is the positive counterpart to
// TestSSEProducerReleasedOnClientDisconnect: events produced after the
// handler has returned must still reach the client. S-19 moved the
// managedStream teardown into writeSuccess's defer, which closes the
// producer's done channel when the handler returns — before fasthttp's
// body-stream writer starts consuming. The producer then exits, streamCh
// closes empty, and the client receives a 200 text/event-stream response
// with zero events (the notification/audit/log/workflow SSE outage).
func TestSSEEventsReachClient(t *testing.T) {
	tr := &HTTPTransport{}

	src := make(chan any) // unbuffered; the test feeds events after connecting
	resp, ok := streamChannel(context.Background(), src, func(v any) any { return v })
	if !ok {
		t.Fatal("streamChannel returned false")
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	srv := &fasthttp.Server{Handler: func(ctx *fasthttp.RequestCtx) {
		tr.writeSuccess(ctx, resp)
	}}
	go func() {
		_ = srv.Serve(ln)
	}()
	defer func() {
		_ = srv.Shutdown()
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("GET / HTTP/1.1\r\nHost: test\r\n\r\n")); err != nil {
		t.Fatalf("write request: %v", err)
	}

	reader := bufio.NewReader(conn)
	// Drain the response head (status line + headers + blank line).
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read response head: %v", err)
		}
		if line == "\r\n" {
			break
		}
	}

	// The stream writer is running by now (fasthttp starts the writer
	// goroutine from SetBodyStreamWriter). If the producer was torn down on
	// handler return, this send blocks forever and the read below times out.
	select {
	case src <- "hello":
	case <-time.After(5 * time.Second):
		t.Fatal("producer was torn down before the stream writer started (S-19 regression)")
	}

	deadline := time.Now().Add(5 * time.Second)
	_ = conn.SetReadDeadline(deadline)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read event (deadline %s): %v", deadline, err)
		}
		if len(line) > len("data: ") && line[:len("data: ")] == "data: " {
			if !strings.Contains(line, `"hello"`) {
				t.Fatalf("unexpected event line: %q", line)
			}
			// End the stream so fasthttp can finish the response and the
			// deferred Shutdown below does not wait on an open connection.
			close(src)
			return
		}
	}
}

// TestSSEDisconnectCancelsHandlerContext proves the second half of the SSE
// lifecycle: when the client disconnects, the writer-owned managedStream.Close
// must cancel the handler context, not just close the producer's done channel.
// Stream services park on ctx.Done() (fasthttp's RequestCtx.Done never fires on
// disconnect), so without this they keep their persistence subscription alive
// after the client is gone — the zombie subscriptions seen in the probe.
func TestSSEDisconnectCancelsHandlerContext(t *testing.T) {
	tr := &HTTPTransport{}

	src := make(chan any) // never closed: the upstream subscription
	ctxCh := make(chan context.Context, 1)
	handler := withStreamLifetime(func(ctx context.Context, req Request) (Response, error) {
		ctxCh <- ctx
		resp, ok := streamChannel(ctx, src, func(v any) any { return v })
		if !ok {
			return Response{}, fmt.Errorf("streamChannel returned false")
		}
		return resp, nil
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	srv := &fasthttp.Server{Handler: func(ctx *fasthttp.RequestCtx) {
		resp, err := handler(ctx, Request{})
		if err != nil {
			t.Errorf("handler: %v", err)
			return
		}
		tr.writeSuccess(ctx, resp)
	}}
	go func() {
		_ = srv.Serve(ln)
	}()
	defer func() {
		_ = srv.Shutdown()
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if _, err := conn.Write([]byte("GET / HTTP/1.1\r\nHost: test\r\n\r\n")); err != nil {
		t.Fatalf("write request: %v", err)
	}

	reader := bufio.NewReader(conn)
	if _, err := reader.ReadString('\n'); err != nil {
		t.Fatalf("read status line: %v", err)
	}
	_ = conn.Close() // client disconnects mid-stream

	var handlerCtx context.Context
	select {
	case handlerCtx = <-ctxCh:
	case <-time.After(5 * time.Second):
		t.Fatal("handler was not invoked")
	}

	// Feed events so the writer attempts a flush against the dead socket.
	deadline := time.Now().Add(10 * time.Second)
	for {
		select {
		case <-handlerCtx.Done():
			if err := handlerCtx.Err(); err != context.Canceled {
				t.Fatalf("handler ctx err = %v, want context.Canceled", err)
			}
			return
		case src <- "event":
		case <-time.After(100 * time.Millisecond):
		}
		if time.Now().After(deadline) {
			t.Fatal("client disconnect never cancelled the handler context (zombie subscription)")
		}
	}
}

// TestSSESurvivesWriteTimeoutAndEmitsKeepalive pins the long-lived-response
// contract: fasthttp applies Server.WriteTimeout as an absolute deadline for
// the whole response, so headerReceived must exempt SSE requests from it.
// The heartbeat keeps idle streams alive through proxies and surfaces dead
// clients as flush errors.
func TestSSESurvivesWriteTimeoutAndEmitsKeepalive(t *testing.T) {
	const serverTimeout = 300 * time.Millisecond
	tr := &HTTPTransport{writeTimeout: serverTimeout}

	src := make(chan any) // idle: no events, only keepalives
	resp, ok := streamChannel(context.Background(), src, func(v any) any { return v })
	if !ok {
		t.Fatal("streamChannel returned false")
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	srv := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			tr.writeSuccess(ctx, resp)
		},
		HeaderReceived: tr.headerReceived,
	}
	go func() {
		_ = srv.Serve(ln)
	}()
	defer func() {
		close(src)
		_ = srv.Shutdown()
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("GET / HTTP/1.1\r\nHost: test\r\nAccept: text/event-stream\r\n\r\n")); err != nil {
		t.Fatalf("write request: %v", err)
	}

	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read response head: %v", err)
		}
		if line == "\r\n" {
			break
		}
	}

	start := time.Now()
	_ = conn.SetReadDeadline(start.Add(3 * time.Second))
	sawKeepalive := false
	for time.Since(start) < 3*serverTimeout {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("stream cut off after %s (WriteTimeout not extended): %v", time.Since(start), err)
		}
		if strings.Contains(line, ": keepalive") {
			sawKeepalive = true
		}
	}
	if !sawKeepalive {
		t.Fatal("idle stream produced no keepalive comment")
	}
}

// TestHeaderReceivedExemptsOnlySSE pins the per-request write-timeout policy.
// SSE requests must not inherit the normal-response deadline; everything else
// must keep it so slow clients stay bounded.
func TestHeaderReceivedExemptsOnlySSE(t *testing.T) {
	tr := &HTTPTransport{}
	cases := []struct {
		name   string
		accept string
		want   time.Duration
	}{
		{"sse", "text/event-stream", 0},
		{"sse-with-charset", "text/event-stream; charset=utf-8", 0},
		{"json", "application/json", serverWriteTimeout},
		{"none", "", serverWriteTimeout},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var h fasthttp.RequestHeader
			if tc.accept != "" {
				h.Set("Accept", tc.accept)
			}
			if got := tr.headerReceived(&h).WriteTimeout; got != tc.want {
				t.Fatalf("WriteTimeout = %s, want %s", got, tc.want)
			}
		})
	}
}

// TestSSEWriterSurvivesDisconnectWithPerIPConn is the regression test for the
// crash the deadline-refresh approach caused: fasthttp wraps connections in a
// perIPConn when MaxConnsPerIP > 0, and perIPConn.Close nils its embedded
// net.Conn. Touching ctx.Conn() from the writer goroutine after a disconnect
// therefore panicked the whole process. The writer must never need the
// connection object, so this stream can idle across several heartbeats after
// the client goes away without crashing.
func TestSSEWriterSurvivesDisconnectWithPerIPConn(t *testing.T) {
	tr := &HTTPTransport{writeTimeout: 40 * time.Millisecond} // heartbeat = 20ms

	src := make(chan any)
	resp, ok := streamChannel(context.Background(), src, func(v any) any { return v })
	if !ok {
		t.Fatal("streamChannel returned false")
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	srv := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			tr.writeSuccess(ctx, resp)
		},
		MaxConnsPerIP:  4,
		HeaderReceived: tr.headerReceived,
	}
	go func() {
		_ = srv.Serve(ln)
	}()
	defer func() {
		close(src)
		_ = srv.Shutdown()
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if _, err := conn.Write([]byte("GET / HTTP/1.1\r\nHost: test\r\nAccept: text/event-stream\r\n\r\n")); err != nil {
		t.Fatalf("write request: %v", err)
	}
	reader := bufio.NewReader(conn)
	if _, err := reader.ReadString('\n'); err != nil {
		t.Fatalf("read status line: %v", err)
	}
	_ = conn.Close()

	// Let several heartbeat ticks fire against the closed connection. A
	// panic here would take down the test binary.
	time.Sleep(10 * tr.writeTimeout)
}
