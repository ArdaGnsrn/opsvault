package scheduler

import (
	"context"
	"log/slog"

	"github.com/robfig/cron/v3"
)

// Scheduler wraps robfig/cron with graceful shutdown support.
type Scheduler struct {
	c   *cron.Cron
	log *slog.Logger
}

func New(log *slog.Logger) *Scheduler {
	return &Scheduler{c: cron.New(), log: log}
}

func (s *Scheduler) Add(expr string, job func(ctx context.Context)) error {
	_, err := s.c.AddFunc(expr, func() {
		job(context.Background())
	})
	return err
}

// Run starts the cron loop and blocks until ctx is cancelled,
// then waits for any running jobs to complete.
func (s *Scheduler) Run(ctx context.Context) {
	s.c.Start()
	s.log.Info("scheduler started")

	<-ctx.Done()
	s.log.Info("scheduler stopping, waiting for running jobs")

	stopCtx := s.c.Stop()
	<-stopCtx.Done()
	s.log.Info("scheduler stopped")
}
