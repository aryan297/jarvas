package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/memory/domain/entity"
)

type MemoryRepository interface {
	Save(ctx context.Context, m *entity.Memory) error
	FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.Memory, int64, error)
	FindByID(ctx context.Context, id, userID uuid.UUID) (*entity.Memory, error)
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateQdrantID(ctx context.Context, id, qdrantID uuid.UUID) error
	RecordAccess(ctx context.Context, id uuid.UUID) error
}
