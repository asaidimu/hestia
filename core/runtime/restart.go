package runtime

import (
	"context"
	"errors"
	"time"

	"github.com/asaidimu/hestia/core/abstract"
)

// ErrRestartRequired is the sentinel returned by operations whose persistent
// state change only becomes active after a process restart: bootstrap
// credential rotation (system:auth:bootstrap:password:set) and the systemd
// self-update swap. Services must never terminate the process themselves
// (audit A-15): they return this sentinel and the host that owns the
// lifecycle interprets it. The stock host (hestia.Application) exits cleanly
// so the supervisor restarts the process with the new state active; embedded
// hosts override via SetupConfig.OnRestartRequired / SystemOptions.
// OnRestartRequired and may defer or replace the restart.
//
// Transports map the sentinel to 503 Service Unavailable so the caller gets
// an honest response instead of a connection reset.
var ErrRestartRequired = errors.New("restart required to activate the change")

// OnRestartFunc consumes a restart-required outcome.
type OnRestartFunc func(err error)

// RestartLink sits outermost in the dispatcher chain and observes outcomes:
// when a dispatch ends with ErrRestartRequired, the host's restart hook is
// invoked exactly once per outcome - after the transport's completion
// callback has run, so the client's response is fully written before the
// process exits. The hook fires asynchronously with a short grace period
// (matching the 300ms convention used by the update apply paths) because
// the fasthttp response is flushed after the handler returns, not inside
// the completion callback.
//
// The Send contract is preserved verbatim: the link never alters results,
// never double-fires onComplete, and surfaces the synchronous rejection
// unchanged.
type RestartLink struct {
	next abstract.Dispatcher
	hook OnRestartFunc
}

func NewRestartLink(hook OnRestartFunc) *RestartLink {
	return &RestartLink{hook: hook}
}

func (l *RestartLink) Wrap(next abstract.Dispatcher) abstract.Dispatcher {
	return &RestartLink{next: next, hook: l.hook}
}

func (l *RestartLink) Send(ctx context.Context, msg abstract.Message, onComplete abstract.CompletionFunc) error {
	if l.hook == nil {
		return l.next.Send(ctx, msg, onComplete)
	}
	guarded := func(ctx context.Context, res *abstract.Result, err error) {
		abstract.Complete(onComplete, ctx, res, err)
		if err != nil && errors.Is(err, ErrRestartRequired) {
			l.fire(err)
		}
	}
	err := l.next.Send(ctx, msg, guarded)
	if err != nil && errors.Is(err, ErrRestartRequired) {
		l.fire(err)
	}
	return err
}

// fire invokes the hook off the dispatch goroutine. By the time it runs the
// transport callback has already completed, so exiting the process here
// cannot truncate the client response.
func (l *RestartLink) fire(err error) {
	go func() {
		time.Sleep(300 * time.Millisecond)
		l.hook(err)
	}()
}
