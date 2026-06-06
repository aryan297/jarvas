package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/workflow/application/dto"
	"github.com/jarvas/backend/internal/modules/workflow/application/port"
	"github.com/jarvas/backend/internal/modules/workflow/domain/entity"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
)

// DAGExecutor is the engine that runs a workflow — defined here to avoid import cycles.
type DAGExecutor interface {
	Execute(ctx context.Context, run *entity.WorkflowRun, def entity.WorkflowDefinition) error
}

type WorkflowService struct {
	wfRepo  port.WorkflowRepository
	runRepo port.RunRepository
	engine  DAGExecutor
}

func NewWorkflowService(
	wfRepo port.WorkflowRepository,
	runRepo port.RunRepository,
	engine DAGExecutor,
) *WorkflowService {
	return &WorkflowService{wfRepo: wfRepo, runRepo: runRepo, engine: engine}
}

func (s *WorkflowService) Create(ctx context.Context, userID uuid.UUID, req dto.CreateWorkflowRequest) (*dto.WorkflowResponse, error) {
	now := time.Now().UTC()
	w := &entity.Workflow{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Status:      entity.WorkflowDraft,
		Definition:  req.Definition,
		Trigger:     entity.TriggerType(req.TriggerType),
		CronExpr:    req.CronExpr,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if w.Trigger == "" {
		w.Trigger = entity.TriggerManual
	}
	if err := s.wfRepo.Save(ctx, w); err != nil {
		return nil, apperrors.Internal(err)
	}
	return toWorkflowDTO(w), nil
}

func (s *WorkflowService) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*dto.WorkflowResponse, int64, error) {
	wfs, total, err := s.wfRepo.FindByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal(err)
	}
	result := make([]*dto.WorkflowResponse, 0, len(wfs))
	for _, w := range wfs {
		result = append(result, toWorkflowDTO(w))
	}
	return result, total, nil
}

func (s *WorkflowService) GetByID(ctx context.Context, id, userID uuid.UUID) (*dto.WorkflowResponse, error) {
	w, err := s.wfRepo.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	return toWorkflowDTO(w), nil
}

func (s *WorkflowService) Update(ctx context.Context, id, userID uuid.UUID, req dto.UpdateWorkflowRequest) (*dto.WorkflowResponse, error) {
	w, err := s.wfRepo.FindByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if req.Name != "" {
		w.Name = req.Name
	}
	if req.Description != "" {
		w.Description = req.Description
	}
	if req.Status != "" {
		w.Status = entity.WorkflowStatus(req.Status)
	}
	if req.Definition != nil {
		w.Definition = *req.Definition
	}
	if req.TriggerType != "" {
		w.Trigger = entity.TriggerType(req.TriggerType)
	}
	if req.CronExpr != "" {
		w.CronExpr = req.CronExpr
	}
	w.UpdatedAt = time.Now().UTC()
	if err := s.wfRepo.Update(ctx, w); err != nil {
		return nil, apperrors.Internal(err)
	}
	return toWorkflowDTO(w), nil
}

func (s *WorkflowService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	if _, err := s.wfRepo.FindByID(ctx, id, userID); err != nil {
		return err
	}
	return s.wfRepo.Delete(ctx, id)
}

// TriggerRun starts a manual run. Async — returns run record immediately.
func (s *WorkflowService) TriggerRun(ctx context.Context, workflowID, userID uuid.UUID, req dto.TriggerRunRequest) (*dto.RunResponse, error) {
	w, err := s.wfRepo.FindByID(ctx, workflowID, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	run := &entity.WorkflowRun{
		ID:             uuid.New(),
		WorkflowID:     w.ID,
		UserID:         userID,
		Status:         entity.RunPending,
		TriggerPayload: req.Payload,
		CreatedAt:      now,
	}
	if err := s.runRepo.Save(ctx, run); err != nil {
		return nil, apperrors.Internal(err)
	}

	// Execute asynchronously.
	def := w.Definition
	go s.engine.Execute(context.Background(), run, def)

	return toRunDTO(run), nil
}

// ListRuns returns paginated runs for a workflow.
func (s *WorkflowService) ListRuns(ctx context.Context, workflowID, userID uuid.UUID, limit, offset int) ([]*dto.RunResponse, int64, error) {
	if _, err := s.wfRepo.FindByID(ctx, workflowID, userID); err != nil {
		return nil, 0, err
	}
	runs, total, err := s.runRepo.FindByWorkflowID(ctx, workflowID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal(err)
	}
	result := make([]*dto.RunResponse, 0, len(runs))
	for _, r := range runs {
		result = append(result, toRunDTO(r))
	}
	return result, total, nil
}

// GetRun returns a single run (used for polling status).
func (s *WorkflowService) GetRun(ctx context.Context, runID, userID uuid.UUID) (*dto.RunResponse, error) {
	run, err := s.runRepo.FindByID(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run.UserID != userID {
		return nil, apperrors.NotFound("run")
	}
	return toRunDTO(run), nil
}

// ── DTO mappers ───────────────────────────────────────────────────────────────

func toWorkflowDTO(w *entity.Workflow) *dto.WorkflowResponse {
	return &dto.WorkflowResponse{
		ID:          w.ID.String(),
		Name:        w.Name,
		Description: w.Description,
		Status:      string(w.Status),
		Definition:  w.Definition,
		TriggerType: string(w.Trigger),
		CronExpr:    w.CronExpr,
		CreatedAt:   w.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   w.UpdatedAt.Format(time.RFC3339),
	}
}

func toRunDTO(r *entity.WorkflowRun) *dto.RunResponse {
	d := &dto.RunResponse{
		ID:         r.ID.String(),
		WorkflowID: r.WorkflowID.String(),
		Status:     string(r.Status),
		Result:     r.Result,
		ErrorMsg:   r.ErrorMsg,
		CreatedAt:  r.CreatedAt.Format(time.RFC3339),
	}
	if r.StartedAt != nil {
		d.StartedAt = r.StartedAt.Format(time.RFC3339)
	}
	if r.CompletedAt != nil {
		d.CompletedAt = r.CompletedAt.Format(time.RFC3339)
	}
	return d
}
