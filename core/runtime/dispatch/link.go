package dispatch

import (
	"context"
	"sync"

	"github.com/asaidimu/hestia/core/abstract"
)

// Link is the template method for dispatcher chain links (audit A-14). The
// Dispatcher.Send contract — "non-nil return = synchronous rejection, the
// completion callback never fires; nil = accepted, the callback fires
// exactly once, even on panic" — used to be re-encoded by hand in every
// link, where nothing enforced it and the next link could double-fire the
// callback or leak goroutines.
//
// A Link encodes the invariant once:
//
//   - PreCheck (optional) runs synchronously: a non-nil error is the
//     Send return value and the completion callback is never invoked;
//   - Observe (optional) wraps the completion callback to observe or
//     annotate the outcome without altering it; observers built with
//     GuardCompletion cannot double-fire;
//   - acceptance forwards to the wrapped dispatcher verbatim.
//
// Links with richer behavior (context transformation, policy lookups) can
// still implement abstract.DispatcherLink directly; this template covers
// the pre-check/observer shape that most links have.
type Link struct {
	next abstract.Dispatcher
	name string
	pre  LinkPreCheck
	obs  LinkObserver
}

// LinkPreCheck performs the link's synchronous gate. Returning non-nil
// rejects the message: no goroutine is started and the completion callback
// never fires.
type LinkPreCheck func(ctx context.Context, msg abstract.Message) error

// LinkObserver wraps the downstream completion callback. Implementations
// must invoke the wrapped callback (GuardCompletion helps) and must not
// alter the result or error.
type LinkObserver func(onComplete abstract.CompletionFunc) abstract.CompletionFunc

func NewLink(name string, pre LinkPreCheck, obs LinkObserver) *Link {
	return &Link{name: name, pre: pre, obs: obs}
}

// Name reports the link's chain name.
func (l *Link) Name() string { return l.name }

func (l *Link) Wrap(next abstract.Dispatcher) abstract.Dispatcher {
	return &Link{next: next, name: l.name, pre: l.pre, obs: l.obs}
}

func (l *Link) Send(ctx context.Context, msg abstract.Message, onComplete abstract.CompletionFunc) error {
	if l.pre != nil {
		if err := l.pre(ctx, msg); err != nil {
			return err
		}
	}
	if l.obs != nil && onComplete != nil {
		onComplete = l.obs(onComplete)
	}
	return l.next.Send(ctx, msg, onComplete)
}

// GuardCompletion wraps a completion callback so it fires exactly once no
// matter how many times a buggy observer might invoke it. The Dispatcher
// contract promises exactly-once delivery; observers that decorate the
// callback use this to keep that promise even under their own panics or
// duplicate invocations.
func GuardCompletion(onComplete abstract.CompletionFunc) abstract.CompletionFunc {
	var once sync.Once
	return func(ctx context.Context, res *abstract.Result, err error) {
		once.Do(func() {
			abstract.Complete(onComplete, ctx, res, err)
		})
	}
}
