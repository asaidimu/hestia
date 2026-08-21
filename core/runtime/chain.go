package runtime

import "github.com/asaidimu/hestia/core/abstract"

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
//	chain.InsertBefore("audit", tenantLink)
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

// @note #review2-20260821-003 issue P2 #review,#api-design : InsertBefore/InsertAfter never set Name on the new link
//
// InsertBefore inserts link before the link named name (no-op if name isn't
// found); InsertAfter is the symmetric case. Both build the new LinkEntry as
// LinkEntry{Link: link} -- Name is left as the zero value "". Once inserted,
// that link can never be targeted again: a later InsertBefore("", ...) would
// silently match the *first* unnamed link ever inserted (since the loop
// compares l.Name == name and "" == "" is true), and Remove("") has the same
// problem. In practice this means only the initial links passed to
// NewDispatcherChain can be reliably referenced afterward; anything added via
// InsertBefore/InsertAfter is a write-once, unaddressable node in the chain.
//
// This matters for anyone building a dynamic middleware chain at runtime
// (e.g. a P2P transport link inserted conditionally) -- there is currently no
// way to remove or re-anchor around it later without keeping a side-channel
// reference to the exact position.
//
// Resolution: change both signatures to InsertBefore(name string, newName
// string, link abstract.DispatcherLink) (or accept a LinkEntry directly) and
// store it as LinkEntry{Name: newName, Link: link}.
func (c *DispatcherChain) InsertBefore(name string, link abstract.DispatcherLink) {
	for i, l := range c.links {
		if l.Name == name {
			c.links = append(c.links[:i], append([]LinkEntry{{Link: link}}, c.links[i:]...)...)
			return
		}
	}
}

// InsertAfter inserts link after the link named name.
// It is a no-op if name is not found.
func (c *DispatcherChain) InsertAfter(name string, link abstract.DispatcherLink) {
	for i, l := range c.links {
		if l.Name == name {
			c.links = append(c.links[:i+1], append([]LinkEntry{{Link: link}}, c.links[i+1:]...)...)
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
