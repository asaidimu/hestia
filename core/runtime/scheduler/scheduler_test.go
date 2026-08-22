package scheduler_test

import (
	"context"
	"testing"
	"time"

	"github.com/asaidimu/hestia/core/runtime/scheduler"
	"go.uber.org/zap"
)

func TestRegisterAndList(t *testing.T) {
	s := scheduler.New(context.Background(), zap.NewNop())

	named := make(chan string, 5)
	s.Register("job:a", "@every 1s", func(ctx context.Context) error {
		named <- "a"
		return nil
	})
	s.Register("job:b", "@every 2s", func(ctx context.Context) error {
		named <- "b"
		return nil
	})

	jobs := s.List()
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}

	got := make(map[string]bool)
	for _, j := range jobs {
		got[j.Name] = true
		if j.Expr == "" {
			t.Errorf("job %q has empty expr", j.Name)
		}
	}
	if !got["job:a"] || !got["job:b"] {
		t.Errorf("missing jobs, got %v", got)
	}
}

func TestSchedulerStartStop(t *testing.T) {
	s := scheduler.New(context.Background(), zap.NewNop())

	ran := make(chan struct{}, 5)
	s.Register("test:run", "@every 1s", func(ctx context.Context) error {
		ran <- struct{}{}
		return nil
	})

	s.Start()
	defer s.Stop()

	select {
	case <-ran:
	case <-time.After(2 * time.Second):
		t.Fatal("job did not run within 2s")
	}

	jobs := s.List()
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Name != "test:run" {
		t.Errorf("Name = %q, want %q", jobs[0].Name, "test:run")
	}
}

func TestJobInfoFields(t *testing.T) {
	s := scheduler.New(context.Background(), zap.NewNop())

	s.Register("test:info", "@every 5m", func(ctx context.Context) error {
		return nil
	})

	jobs := s.List()
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}

	j := jobs[0]
	if j.Name != "test:info" {
		t.Errorf("Name = %q, want %q", j.Name, "test:info")
	}
	if j.Expr != "@every 5m" {
		t.Errorf("Expr = %q, want %q", j.Expr, "@every 5m")
	}
	if j.Paused {
		t.Error("new job should not be paused")
	}
	if len(j.Tags) == 0 {
		t.Error("expected non-empty tags")
	}
}
