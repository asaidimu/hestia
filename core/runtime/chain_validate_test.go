package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/asaidimu/hestia/core/abstract"
)

// stubLink records its name when Send passes through it.
type stubLink struct {
	name string
}

func (l *stubLink) Wrap(next abstract.Dispatcher) abstract.Dispatcher { return next }

type validateNoopDispatcher struct{}

func (validateNoopDispatcher) Send(ctx context.Context, msg abstract.Message, onComplete abstract.CompletionFunc) error {
	return nil
}

func canonicalTestChain() *DispatcherChain {
	return NewDispatcherChain(
		LinkEntry{Name: "sanitization", Link: &stubLink{name: "sanitization"}},
		LinkEntry{Name: "bootstrap", Link: &stubLink{name: "bootstrap"}},
		LinkEntry{Name: "secure", Link: &stubLink{name: "secure"}},
		LinkEntry{Name: "ratelimit", Link: &stubLink{name: "ratelimit"}},
		LinkEntry{Name: "throttle", Link: &stubLink{name: "throttle"}},
		LinkEntry{Name: "tenant", Link: &stubLink{name: "tenant"}},
		LinkEntry{Name: "blob", Link: &stubLink{name: "blob"}},
		LinkEntry{Name: "audit", Link: &stubLink{name: "audit"}},
	)
}

func TestCanonicalChainValidates(t *testing.T) {
	if err := canonicalTestChain().Validate(); err != nil {
		t.Fatalf("canonical chain must validate, got: %v", err)
	}
}

func TestValidateRejectsRemovedSecurityLink(t *testing.T) {
	c := canonicalTestChain()
	c.Remove("secure")
	err := c.Validate()
	if err == nil {
		t.Fatal("removing the secure link must fail validation")
	}
	if !strings.Contains(err.Error(), "secure") {
		t.Fatalf("error should name the missing link, got: %v", err)
	}
}

func TestValidateRejectsReorderedSecurityLinks(t *testing.T) {
	// secure behind ratelimit: rejected requests would still have consumed
	// rate-limit tokens — the exact regression A-2 guards against.
	c := NewDispatcherChain(
		LinkEntry{Name: "sanitization", Link: &stubLink{}},
		LinkEntry{Name: "bootstrap", Link: &stubLink{}},
		LinkEntry{Name: "ratelimit", Link: &stubLink{}},
		LinkEntry{Name: "secure", Link: &stubLink{}},
		LinkEntry{Name: "throttle", Link: &stubLink{}},
		LinkEntry{Name: "tenant", Link: &stubLink{}},
		LinkEntry{Name: "blob", Link: &stubLink{}},
		LinkEntry{Name: "audit", Link: &stubLink{}},
	)
	err := c.Validate()
	if err == nil {
		t.Fatal("reordering secure behind ratelimit must fail validation")
	}
}

func TestValidateRejectsDuplicateNames(t *testing.T) {
	c := canonicalTestChain()
	c.InsertAfter("audit", "extra", &stubLink{})
	// simulate a duplicate by inserting a canonical name again
	c.InsertAfter("extra", "audit", &stubLink{})
	if err := c.Validate(); err == nil {
		t.Fatal("duplicate canonical names must fail validation")
	}
}

func TestValidateAcceptsNewLinksAnywhere(t *testing.T) {
	c := canonicalTestChain()
	c.InsertBefore("audit", "idempotency", &stubLink{})
	c.InsertAfter("ratelimit", "circuitbreaker", &stubLink{})
	if err := c.Validate(); err != nil {
		t.Fatalf("new links at declared positions must validate, got: %v", err)
	}
}

func TestGuardedEditorRefusesRemovalOfCanonicalLinks(t *testing.T) {
	c := canonicalTestChain()
	g := NewGuardedChainEditor(c)
	g.Remove("secure")
	if len(g.Violations()) == 0 {
		t.Fatal("Remove(secure) must record a violation")
	}
	if !c.has("secure") {
		t.Fatal("guarded Remove must not mutate the chain")
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("chain must still validate after refused removal: %v", err)
	}
}

func TestGuardedEditorRefusesCanonicalNameCollisions(t *testing.T) {
	c := canonicalTestChain()
	g := NewGuardedChainEditor(c)
	g.InsertBefore("audit", "secure", &stubLink{})
	if len(g.Violations()) == 0 {
		t.Fatal("inserting a link named 'secure' must record a violation")
	}
}

func TestGuardedEditorAllowsNewLinks(t *testing.T) {
	c := canonicalTestChain()
	g := NewGuardedChainEditor(c)
	g.InsertBefore("audit", "idempotency", &stubLink{})
	if len(g.Violations()) != 0 {
		t.Fatalf("new links must be allowed, violations: %v", g.Violations())
	}
	// the guarded editor edits the wrapped chain in place
	if !c.has("idempotency") {
		t.Fatal("inserted link missing from chain")
	}
}
