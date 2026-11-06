package di

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type concurrentDep struct{}

type concurrentSvc struct{ dep concurrentDep }

// TestConcurrentTransientResolutionDoesNotReportCycle guards against sharing
// the dependency-cycle tracking across goroutines. Concurrent resolutions of
// the same transient provider used to see each other's in-flight type and fail
// with a bogus "dependency cycle detected" error.
func TestConcurrentTransientResolutionDoesNotReportCycle(t *testing.T) {
	const workers = 8

	var armed atomic.Bool
	var ready sync.WaitGroup
	gate := make(chan struct{})
	ready.Add(workers)

	c := New()
	if err := c.Register(func(*Container) (concurrentDep, error) {
		return concurrentDep{}, nil
	}, WithLifetime(Transient)); err != nil {
		t.Fatal(err)
	}
	if err := c.Register(func(c *Container) (concurrentSvc, error) {
		if armed.Load() {
			ready.Done()
			<-gate
		}
		d, err := c.Resolve[concurrentDep]()
		if err != nil {
			return concurrentSvc{}, err
		}
		return concurrentSvc{dep: d}, nil
	}, WithLifetime(Transient)); err != nil {
		t.Fatal(err)
	}

	if err := c.Build(); err != nil {
		t.Fatalf("build: %v", err)
	}
	armed.Store(true)

	// Release the workers once all of them are inside the constructor, or
	// after a timeout so a regression fails instead of hanging the suite.
	released := make(chan struct{})
	go func() {
		done := make(chan struct{})
		go func() {
			ready.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
		close(gate)
		close(released)
	}()

	errs := make([]error, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := range workers {
		go func(i int) {
			defer wg.Done()
			_, errs[i] = c.Resolve[concurrentSvc]()
		}(i)
	}
	wg.Wait()
	<-released

	for i, err := range errs {
		if err != nil {
			t.Errorf("worker %d: %v", i, err)
		}
	}
}

type cycleA struct{}

type cycleB struct{}

// TestBuildDetectsCycle verifies that graph validation still catches cycles
// even though runtime resolution no longer tracks in-flight types.
func TestBuildDetectsCycle(t *testing.T) {
	c := New()
	if err := c.Register(func(c *Container) (cycleA, error) {
		_, err := c.Resolve[cycleB]()
		return cycleA{}, err
	}); err != nil {
		t.Fatal(err)
	}
	if err := c.Register(func(c *Container) (cycleB, error) {
		_, err := c.Resolve[cycleA]()
		return cycleB{}, err
	}); err != nil {
		t.Fatal(err)
	}

	if err := c.Build(); err == nil {
		t.Fatal("expected a dependency cycle error")
	}
}
