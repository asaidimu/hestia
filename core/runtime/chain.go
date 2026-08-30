package runtime

import (
	"fmt"
	"slices"

	"github.com/asaidimu/hestia/core/abstract"
)

// DefaultChainOrder is the canonical, security-relevant composition order of
// the dispatcher chain (audit A-2). One HTTP message crosses exactly these
// links, outermost first:
//
//	sanitization → bootstrap → secure → ratelimit → throttle → tenant → blob → audit → terminal
//
// The order is load-bearing:
//
//   - sanitization scopes and sanitizes outgoing documents for everything
//     downstream, so it must wrap the whole chain.
//   - bootstrap gates un-bootstrapped deployments before any policy
//     evaluation happens.
//   - secure (authorization) runs BEFORE ratelimit/throttle deliberately:
//     rejected requests must not consume rate-limit tokens.
//   - tenant scopes persistence after authorization but before any handler
//     executes; audit runs innermost so it records the post-tenant,
//     post-blob picture of the request that the terminal actually sees.
//
// The order used to be encoded in exactly one function (and silently
// mutable by embedders via DispatcherChainFunc). It is now validated data:
// SystemModule.DispatcherChain runs Validate after embedder mutations and
// fails closed — an invalid chain is discarded and the canonical order is
// built instead (see GuardedChainEditor for the editor embedders receive,
// which records violations instead of applying them silently).
var DefaultChainOrder = []string{
	"sanitization",
	"bootstrap",
	"secure",
	"ratelimit",
	"throttle",
	"tenant",
	"blob",
	"audit",
}

// LinkEntry pairs a name with its DispatcherLink for position-based
// insertion and removal.
type LinkEntry struct {
	Name string
	Link abstract.DispatcherLink
}

// DispatcherChain is an ordered, mutable list of DispatcherLinks.
// Build composes them around a base Dispatcher (outermost first).
//
//	chain := NewDispatcherChain(
//	    LinkEntry{"secure", secureLink},
//	    LinkEntry{"audit",  auditLink},
//	)
//	chain.InsertBefore("audit", "tenant", tenantLink)
//	disp := chain.Build(baseDispatcher)
//	// result: secure → tenant → audit → base
type DispatcherChain struct {
	links []LinkEntry
}

func NewDispatcherChain(entries ...LinkEntry) *DispatcherChain {
	c := &DispatcherChain{}
	c.links = append(c.links, entries...)
	return c
}

// InsertBefore inserts a named link before the link named name.
// It is a no-op if name is not found.
func (c *DispatcherChain) InsertBefore(name string, newName string, link abstract.DispatcherLink) {
	for i, l := range c.links {
		if l.Name == name {
			c.links = append(c.links[:i], append([]LinkEntry{{Name: newName, Link: link}}, c.links[i:]...)...)
			return
		}
	}
}

// InsertAfter inserts a named link after the link named name.
// It is a no-op if name is not found.
func (c *DispatcherChain) InsertAfter(name string, newName string, link abstract.DispatcherLink) {
	for i, l := range c.links {
		if l.Name == name {
			c.links = append(c.links[:i+1], append([]LinkEntry{{Name: newName, Link: link}}, c.links[i+1:]...)...)
			return
		}
	}
}

// Remove removes the link with the given name.
// It is a no-op if name is not found.
func (c *DispatcherChain) Remove(name string) {
	for i, l := range c.links {
		if l.Name == name {
			c.links = append(c.links[:i], c.links[i+1:]...)
			return
		}
	}
}

// Build wraps base with all links. The first link in the chain
// becomes the outermost wrapper, so the execution order is:
//
//	links[0] → links[1] → … → links[n-1] → base
func (c *DispatcherChain) Build(base abstract.Dispatcher) abstract.Dispatcher {
	d := base
	for i := len(c.links) - 1; i >= 0; i-- {
		d = c.links[i].Link.Wrap(d)
	}
	return d
}

// has reports whether a link with the given name is present.
func (c *DispatcherChain) has(name string) bool {
	for _, l := range c.links {
		if l.Name == name {
			return true
		}
	}
	return false
}

// Validate checks the chain against the canonical composition order:
//
//   - every name must be unique;
//   - every security-critical canonical link must still be present
//     (removal is a violation, not a no-op);
//   - the relative order of canonical links must be preserved — embedders
//     may insert new links anywhere, but must not move secure behind
//     ratelimit or audit out of its terminal-observer position.
//
// A nil return guarantees the chain is safe to Build.
func (c *DispatcherChain) Validate() error {
	seen := make(map[string]bool, len(c.links))
	last := -1
	for _, l := range c.links {
		if seen[l.Name] {
			return fmt.Errorf("dispatcher chain: duplicate link name %q", l.Name)
		}
		seen[l.Name] = true
		if idx := slices.Index(DefaultChainOrder, l.Name); idx >= 0 {
			if idx == last {
				return fmt.Errorf("dispatcher chain: link %q appears twice", l.Name)
			}
			if idx < last {
				return fmt.Errorf("dispatcher chain: security-critical link %q moved out of canonical order (position %d, expected after position %d)", l.Name, idx, last)
			}
			last = idx
		}
	}
	for _, name := range DefaultChainOrder {
		if !seen[name] {
			return fmt.Errorf("dispatcher chain: missing security-critical link %q", name)
		}
	}
	return nil
}

// GuardedChainEditor is the ChainEditor embedders receive through
// SystemOptions.DispatcherChainFunc (audit A-2). It enforces the extension
// contract at mutation time instead of trusting the embedder:
//
//   - canonical links cannot be removed (Remove records a violation and
//     leaves the chain untouched);
//   - new links must not shadow canonical names;
//   - inserts at any position are allowed — Validate still re-checks the
//     final composition.
type GuardedChainEditor struct {
	chain      *DispatcherChain
	violations []string
}

func NewGuardedChainEditor(chain *DispatcherChain) *GuardedChainEditor {
	return &GuardedChainEditor{chain: chain}
}

// Violations returns the contract violations recorded so far.
func (g *GuardedChainEditor) Violations() []string {
	return g.violations
}

func (g *GuardedChainEditor) InsertBefore(name string, newName string, link abstract.DispatcherLink) {
	if slices.Contains(DefaultChainOrder, newName) {
		g.violations = append(g.violations, fmt.Sprintf("InsertBefore %q: name collides with a security-critical canonical link", newName))
		return
	}
	g.chain.InsertBefore(name, newName, link)
}

func (g *GuardedChainEditor) InsertAfter(name string, newName string, link abstract.DispatcherLink) {
	if slices.Contains(DefaultChainOrder, newName) {
		g.violations = append(g.violations, fmt.Sprintf("InsertAfter %q: name collides with a security-critical canonical link", newName))
		return
	}
	g.chain.InsertAfter(name, newName, link)
}

// Remove of a canonical (security-critical) link is refused and recorded;
// removal of embedder-added links is allowed.
func (g *GuardedChainEditor) Remove(name string) {
	if slices.Contains(DefaultChainOrder, name) {
		g.violations = append(g.violations, fmt.Sprintf("Remove %q: security-critical links cannot be removed", name))
		return
	}
	g.chain.Remove(name)
}
