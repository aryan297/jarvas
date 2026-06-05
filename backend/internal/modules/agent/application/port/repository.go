package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/agent/domain/entity"
)

type AgentRepository interface {
	Save(ctx context.Context, a *entity.Agent) error
	FindByID(ctx context.Context, id, userID uuid.UUID) (*entity.Agent, error)
	FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.Agent, int64, error)
	Update(ctx context.Context, a *entity.Agent) error
	Delete(ctx context.Context, id uuid.UUID) error
}
