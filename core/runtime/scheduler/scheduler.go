package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	cron "github.com/netresearch/go-cron"
	"go.uber.org/zap"
)

type JobInfo struct {
	Name     string    `json:"name"`
	Expr     string    `json:"expr"`
	Next     time.Time `json:"next"`
	Prev     time.Time `json:"prev"`
	Paused   bool      `json:"paused"`
	Tags     []string  `json:"tags,omitempty"`
}

type Scheduler struct {
	c      *cron.Cron
	logger *zap.Logger
	mu     sync.RWMutex
	exprs  map[string]string // name -> cron expression
}

func New(logger *zap.Logger) *Scheduler {
	return &Scheduler{
		c: cron.New(cron.WithChain(
			cron.Recover(cron.DefaultLogger),
			cron.SkipIfStillRunning(cron.DefaultLogger),
		)),
		logger: logger,
		exprs:  make(map[string]string),
	}
}

func (s *Scheduler) Register(name, expr string, fn func(context.Context) error) {
	_, err := s.c.AddFunc(expr, func() { fn(context.Background()) },
		cron.WithName(name),
		cron.WithTags("hestia"),
	)
	if err != nil {
		s.logger.Warn("scheduler: failed to register job", zap.String("name", name), zap.String("expr", expr), zap.Error(err))
		return
	}
	s.mu.Lock()
	s.exprs[name] = expr
	s.mu.Unlock()
	s.logger.Info("scheduler: registered job", zap.String("name", name), zap.String("expr", expr))
}

func (s *Scheduler) List() []JobInfo {
	entries := s.c.Entries()
	info := make([]JobInfo, 0, len(entries))
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range entries {
		expr, _ := s.exprs[e.Name]
		info = append(info, JobInfo{
			Name:   e.Name,
			Expr:   expr,
			Next:   e.Next,
			Prev:   e.Prev,
			Paused: e.Paused,
			Tags:   e.Tags,
		})
	}
	return info
}

func (s *Scheduler) Start() {
	s.c.Start()
	s.logger.Info("scheduler: started")
}

func (s *Scheduler) Stop() {
	<-s.c.Stop().Done()
	s.logger.Info("scheduler: stopped")
}

func (s *Scheduler) String() string {
	return fmt.Sprintf("Scheduler{jobs=%d}", len(s.c.Entries()))
}
