package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

type Scheduler struct {
	rootCtx context.Context
	wg      sync.WaitGroup

	logger *slog.Logger
}

func New(rootCtx context.Context, logger *slog.Logger) *Scheduler {
	return &Scheduler{
		logger:  logger,
		rootCtx: rootCtx,
	}
}

func (s *Scheduler) Schedule(fn func(context.Context) error, report func(context.Context, error) error) {
	s.wg.Go(func() {
		if err := fn(s.rootCtx); err != nil {
			if rErr := report(s.rootCtx, err); rErr != nil {
				s.logger.Error(fmt.Sprintf("failed to report an error (%v): %v", rErr, err))
			}
		}
	})
}

func (s *Scheduler) Wait() {
	s.wg.Wait()
}
