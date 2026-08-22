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
