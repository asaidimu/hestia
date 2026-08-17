// Package di is a light, generic dependency injection container for Go.
//
// Design summary:
//
//   - Providers are registered against a reflect.Type, derived automatically
//     from the generic type parameter passed to Register/RegisterInstance.
//
//   - A constructor is a function that accepts the container/registry and
//     returns the constructed instance: func(c *Container) (T, error) or
//     func(c *Container) T. Dependencies are resolved explicitly within the
//     constructor via di.Get[Dep](c) or c.Get[Dep]().
//
//   - Build() performs a full graph pass before application runtime:
//     depth-first traversal eagerly constructs singletons and validates transient
//     dependencies, detecting cycles (with a readable trail) and missing providers.
//
//   - Two lifetimes are supported: Singleton (default; built once at Build()
//     time and cached) and Transient (constructed fresh on every Get()).
package di

import (
	"cmp"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"sync"
)

// Lifetime controls how many times a provider's constructor is invoked.
type Lifetime int

const (
	// Singleton providers are constructed once, during Build(), and the
	// same instance is returned from every subsequent Get().
	Singleton Lifetime = iota
	// Transient providers are constructed fresh on every Get() call.
	Transient
)

func (l Lifetime) String() string {
	switch l {
	case Singleton:
		return "singleton"
	case Transient:
		return "transient"
	default:
		return "unknown"
	}
}

// Option configures a provider at registration time.
type Option func(*provider)

// WithLifetime sets the lifetime for a provider. The default is Singleton.
func WithLifetime(l Lifetime) Option {
	return func(p *provider) { p.lifetime = l }
}

// Constructor is a function that accepts the container/registry and returns T and an optional error.
type Constructor[T any] func(c *Container) (T, error)

type provider struct {
	typ      reflect.Type
	ctor     func(c *Container) (any, error)
	lifetime Lifetime

	built    bool
	instance any
}

// Container holds provider registrations and, once Build() has been called,
// the validated dependency graph and constructed singletons.
type Container struct {
	mu        sync.Mutex
	providers map[reflect.Type]*provider
	resolving map[reflect.Type]bool
	stack     []reflect.Type
	building  bool
	built     bool
}

// New creates an empty, unbuilt Container.
func New() *Container {
	return &Container{
		providers: make(map[reflect.Type]*provider),
		resolving: make(map[reflect.Type]bool),
	}
}

// Register registers constructor as a method on Container for type T.
func (c *Container) Register[T any](ctor func(*Container) (T, error), opts ...Option) error {
	return c.RegisterType(reflect.TypeFor[T](), func(c *Container) (any, error) {
		return ctor(c)
	}, opts...)
}

// RegisterType registers a constructor keyed by an explicit reflect.Type.
// It is the non-generic seam behind Register so concrete containers can
// implement an abstract registration contract.
func (c *Container) RegisterType(t reflect.Type, ctor func(*Container) (any, error), opts ...Option) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.built {
		return fmt.Errorf("di: Register(%s): container is already built", t)
	}

	if _, exists := c.providers[t]; exists {
		return fmt.Errorf("di: Register(%s): type already registered", t)
	}

	p := &provider{
		typ:      t,
		ctor:     ctor,
		lifetime: Singleton,
	}
	for _, opt := range opts {
		opt(p)
	}

	c.providers[t] = p
	return nil
}

// RegisterInstance registers a pre-built value as a singleton provider for type T.
func RegisterInstance[T any](c *Container, value T) error {
	return c.RegisterInstance(value)
}

// RegisterInstance registers a pre-built value as a singleton provider on Container for type T.
func (c *Container) RegisterInstance[T any](value T) error {
	return c.RegisterInstanceType(reflect.TypeFor[T](), value)
}

// RegisterInstanceType registers a pre-built value as a singleton provider
// keyed by an explicit reflect.Type. Non-generic seam behind RegisterInstance.
func (c *Container) RegisterInstanceType(t reflect.Type, value any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.built {
		return fmt.Errorf("di: RegisterInstance(%s): container is already built", t)
	}

	if _, exists := c.providers[t]; exists {
		return fmt.Errorf("di: RegisterInstance(%s): type already registered", t)
	}

	c.providers[t] = &provider{
		typ:      t,
		lifetime: Singleton,
		built:    true,
		instance: value,
	}
	return nil
}

// Build validates the full dependency graph: executes singletons eagerly and
// dry-runs transients to verify missing dependencies and detect dependency cycles.
func (c *Container) Build() error {
	c.mu.Lock()
	if c.built {
		c.mu.Unlock()
		return fmt.Errorf("di: Build: container is already built")
	}
	c.building = true
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		c.building = false
		c.mu.Unlock()
	}()

	// Visit in deterministic order (sorted by type name)
	c.mu.Lock()
	types := slices.Collect(maps.Keys(c.providers))
	c.mu.Unlock()

	slices.SortFunc(types, func(a, b reflect.Type) int {
		return cmp.Compare(a.String(), b.String())
	})

	for _, t := range types {
		if _, err := c.resolve(t); err != nil {
			return err
		}
	}

	c.mu.Lock()
	c.built = true
	c.mu.Unlock()
	return nil
}

// Rebuild allows adding new providers post-build and re-verifying the graph.
func (c *Container) Rebuild() error {
	c.mu.Lock()
	c.built = false
	c.mu.Unlock()

	return c.Build()
}

// resolve constructs (or returns the cached instance of) provider for t.
func (c *Container) resolve(t reflect.Type) (any, error) {
	c.mu.Lock()
	p, ok := c.providers[t]
	if !ok {
		reqBy := ""
		if len(c.stack) > 0 {
			reqBy = fmt.Sprintf(" (required by %s)", c.stack[len(c.stack)-1])
		}
		c.mu.Unlock()
		return nil, fmt.Errorf("di: no provider registered for %s%s", t, reqBy)
	}

	if p.built {
		val := p.instance
		c.mu.Unlock()
		return val, nil
	}

	if c.resolving[t] {
		trail := append(slices.Clone(c.stack), t)
		c.mu.Unlock()
		return nil, fmt.Errorf("di: Build: dependency cycle detected: %s", formatTrail(trail))
	}

	c.resolving[t] = true
	c.stack = append(c.stack, t)
	c.mu.Unlock()

	// Execute constructor without lock so nested c.Get calls don't deadlock
	val, err := p.ctor(c)

	c.mu.Lock()
	delete(c.resolving, t)
	c.stack = c.stack[:len(c.stack)-1]

	if err != nil {
		c.mu.Unlock()
		return nil, fmt.Errorf("di: constructing %s: %w", t, err)
	}

	if p.lifetime == Singleton {
		p.instance = val
		p.built = true
	}
	c.mu.Unlock()

	return val, nil
}

// Get resolves and returns the value registered for T as a Container method.
func (c *Container) Resolve[T any]() (T, error) {
	var zero T
	t := reflect.TypeFor[T]()

	v, err := c.ResolveType(t)
	if err != nil {
		return zero, err
	}

	out, ok := v.(T)
	if !ok {
		return zero, fmt.Errorf("di: Get(%s): resolved value of type %T is not assignable to requested type", t, v)
	}
	return out, nil
}

// ResolveType resolves and returns the value registered for t as an any.
// Non-generic seam behind Resolve so concrete containers can implement an
// abstract resolution contract.
func (c *Container) ResolveType(t reflect.Type) (any, error) {
	c.mu.Lock()
	canGet := c.built || c.building
	if !canGet {
		if p, ok := c.providers[t]; ok && p.built {
			// Pre-registered instances are resolvable before Build.
			canGet = true
		}
	}
	c.mu.Unlock()

	if !canGet {
		return nil, fmt.Errorf("di: Get(%s): container has not been Build() yet", t)
	}

	return c.resolve(t)
}

// MustResolve is like Resolve but panics on error.
func (c *Container) MustResolve[T any]() T {
	v, err := c.Resolve[T]()
	if err != nil {
		panic(err)
	}
	return v
}

func formatTrail(trail []reflect.Type) string {
	parts := make([]string, len(trail))
	for i, t := range trail {
		parts[i] = t.String()
	}
	return strings.Join(parts, " -> ")
}
