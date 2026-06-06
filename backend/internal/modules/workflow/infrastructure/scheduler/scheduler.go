package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"

	"github.com/jarvas/backend/internal/modules/workflow/application/port"
	"github.com/jarvas/backend/internal/modules/workflow/application/service"
	"github.com/jarvas/backend/internal/modules/workflow/application/dto"
	"github.com/jarvas/backend/internal/shared/logger"
	"go.uber.org/zap"
)

// Scheduler loads ACTIVE+SCHEDULE workflows on startup and fires them on cron.
type Scheduler struct {
	cron    *cron.Cron
	wfRepo  port.WorkflowRepository
	wfSvc   *service.WorkflowService
	entries map[uuid.UUID]cron.EntryID
}

func New(wfRepo port.WorkflowRepository, wfSvc *service.WorkflowService) *Scheduler {
	return &Scheduler{
		cron:    cron.New(cron.WithSeconds()),
		wfRepo:  wfRepo,
		wfSvc:   wfSvc,
		entries: make(map[uuid.UUID]cron.EntryID),
	}
}

// Start loads existing scheduled workflows and begins the cron loop.
func (s *Scheduler) Start(ctx context.Context) error {
	workflows, err := s.wfRepo.FindActiveScheduled(ctx)
	if err != nil {
		return fmt.Errorf("scheduler load: %w", err)
	}

	for _, wf := range workflows {
		if wf.CronExpr != "" {
			if err := s.add(wf.ID, wf.UserID, wf.CronExpr); err != nil {
				logger.Warn("scheduler: could not add workflow",
					zap.String("workflow_id", wf.ID.String()),
					zap.Error(err))
			}
		}
	}

	s.cron.Start()
	logger.Info("workflow scheduler started", zap.Int("jobs", len(s.entries)))
	return nil
}

// Stop shuts down the cron gracefully.
func (s *Scheduler) Stop() {
	s.cron.Stop()
}

// Add registers a new cron job for a workflow.
func (s *Scheduler) add(workflowID, userID uuid.UUID, cronExpr string) error {
	wfID := workflowID
	uID := userID

	entryID, err := s.cron.AddFunc(cronExpr, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		_, err := s.wfSvc.TriggerRun(ctx, wfID, uID, dto.TriggerRunRequest{
			Payload: map[string]interface{}{"trigger": "schedule"},
		})
		if err != nil {
			logger.Error("scheduled workflow run failed",
				zap.String("workflow_id", wfID.String()),
				zap.Error(err))
		}
	})
	if err != nil {
		return fmt.Errorf("cron add %q: %w", cronExpr, err)
	}
	s.entries[workflowID] = entryID
	return nil
}
