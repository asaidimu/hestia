package dispatch

import (
        "context"

        "github.com/asaidimu/go-anansi/v8/core/data"

        "github.com/asaidimu/hestia/core/abstract"
)

type DispatchInput struct {
        Name     string
        Context  context.Context
        ID       string
        Document data.Documenter
        // DocumentStream feeds real streamed input to the handler: one
        // StreamItem per NDJSON body item, produced (and closed) by the
        // transport's body decoder. Mutually exclusive with Document; when set,
        // the message's InputChannel() returns this channel verbatim and the
        // producer owns its lifecycle. Mutually exclusive with the Stream-intent
        // readiness barrier below.
        DocumentStream <-chan abstract.StreamItem
        Intent         abstract.Verb
}

const streamChannelBuffer = 1

// Await dispatches msg and blocks until it completes or ctx expires.
//
// A non-nil error is either a synchronous rejection by the dispatcher chain
// (handler missing, unauthorized, rate-limited — delivered before any
// goroutine is started) or ctx expiry while waiting; disambiguate with
// errors.Is(err, context.Canceled | context.DeadlineExceeded).
func Await(ctx context.Context, d abstract.Dispatcher, msg abstract.Message) (*abstract.Result, error) {
        ch := make(chan awaitOutcome, 1)
        if err := d.Send(ctx, msg, func(_ context.Context, res *abstract.Result, err error) {
                ch <- awaitOutcome{res: res, err: err}
        }); err != nil {
                return nil, err
        }
        select {
        case out := <-ch:
                return out.res, out.err
        case <-ctx.Done():
                return nil, ctx.Err()
        }
}

type awaitOutcome struct {
        res *abstract.Result
        err error
}

// newMessageFrom builds the message described by in. Three input shapes:
// real streamed input (DocumentStream — producer-owned channel, forwarded
// verbatim), the Stream-intent readiness barrier (buffered channel, fed after
// completion, matching the historical synchronous ordering), or a single
// document. The second return is the writable signal channel (non-nil only
// for the readiness-barrier shape; Dispatch feeds it after completion).
func newMessageFrom(in DispatchInput) (*message, chan abstract.StreamItem) {
        msgID := in.ID
        if msgID == "" {
                msgID = MustNewID()
        }

        var signalCh chan abstract.StreamItem
        switch {
        case in.DocumentStream != nil:
                // Real streamed input: the producer (HTTP transport's body decoder)
                // owns the channel and its lifecycle, including closing it when the
                // body ends or errors. Dispatch just forwards it.
                signalCh = nil
        case in.Intent == abstract.Stream:
                // Existing output-only readiness barrier — unchanged behavior,
                // re-typed to StreamItem.
                signalCh = make(chan abstract.StreamItem, streamChannelBuffer)
        }

        msg := &message{
                id:      msgID,
                name:    in.Name,
                ctx:     in.Context,
                input:   in.Document,
                inputCh: signalCh,
        }
        if in.DocumentStream != nil {
                msg.inputCh = in.DocumentStream
        }
        return msg, signalCh
}

// Dispatch builds a message from in and awaits its completion. For Stream
// intents the start signal is fed into the message's input channel after the
// handler has run, matching the historical synchronous ordering.
func Dispatch(ctx context.Context, disp abstract.Dispatcher, in DispatchInput) (*abstract.Result, error) {
        msg, signalCh := newMessageFrom(in)

        result, err := Await(ctx, disp, msg)
        if err != nil {
                return nil, err
        }

        if signalCh != nil {
                // The channel is buffered(1): this never blocks even though the
                // handler has already returned.
                signalCh <- abstract.StreamItem{}
                close(signalCh)
        }

        return result, nil
}

// Enqueue accepts a message for asynchronous execution and returns
// immediately with its correlation ID. It is the fire-and-forget counterpart
// of Dispatch: handler outcome and panics are discarded.
//
// err is non-nil only for synchronous rejections (unauthorized, rate-limited,
// handler missing); on nil err the message was accepted for execution but its
// completion is not awaited. Until durable execution exists, acceptance is
// best-effort: if the process exits before the handler runs, the work is lost.
func Enqueue(ctx context.Context, disp abstract.Dispatcher, in DispatchInput) (string, error) {
        msg, _ := newMessageFrom(in)
        if err := disp.Send(ctx, msg, nil); err != nil {
                return "", err
        }
        return msg.ID(), nil
}
