package port

import (
	"context"

	"github.com/google/uuid"
	toolentity "github.com/jarvas/backend/internal/modules/tool/domain/entity"
)

type ToolRepository interface {
	FindAll(ctx context.Context) ([]*toolentity.Tool, error)
	FindByName(ctx context.Context, name string) (*toolentity.Tool, error)
}

type UserToolConfigRepository interface {
	Save(ctx context.Context, cfg *toolentity.UserToolConfig) error
	FindByUserAndTool(ctx context.Context, userID, toolID uuid.UUID) (*toolentity.UserToolConfig, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*toolentity.UserToolConfig, error)
}
