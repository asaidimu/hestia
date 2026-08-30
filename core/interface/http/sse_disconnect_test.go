package http

import (
	"bufio"
	"context"
	"fmt"
	"net"
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
	// The producer must have exited: an unbuffered send finds no receiver.
	select {
	case src <- "probe":
		t.Fatal("producer still consuming after client disconnect — goroutine leak (S-19)")
	case <-time.After(2 * time.Second):
	}
}
