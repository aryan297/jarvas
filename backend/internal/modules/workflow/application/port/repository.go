package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/workflow/domain/entity"
)

type WorkflowRepository interface {
	Save(ctx context.Context, w *entity.Workflow) error
	FindByID(ctx context.Context, id, userID uuid.UUID) (*entity.Workflow, error)
	FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.Workflow, int64, error)
	FindActiveScheduled(ctx context.Context) ([]*entity.Workflow, error)
	Update(ctx context.Context, w *entity.Workflow) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type RunRepository interface {
	Save(ctx context.Context, r *entity.WorkflowRun) error
	Update(ctx context.Context, r *entity.WorkflowRun) error
	FindByWorkflowID(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*entity.WorkflowRun, int64, error)
	FindByID(ctx context.Context, id uuid.UUID) (*entity.WorkflowRun, error)
}
