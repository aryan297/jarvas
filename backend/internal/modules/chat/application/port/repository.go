package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/chat/domain/entity"
)

type ConversationRepository interface {
	Create(ctx context.Context, conv *entity.Conversation) error
	FindByID(ctx context.Context, id, userID uuid.UUID) (*entity.Conversation, error)
	FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.Conversation, int64, error)
	UpdateTitle(ctx context.Context, id uuid.UUID, title string) error
	Archive(ctx context.Context, id uuid.UUID) error
}

type MessageRepository interface {
	Save(ctx context.Context, msg *entity.Message) error
	FindByConversationID(ctx context.Context, convID uuid.UUID, limit, offset int) ([]*entity.Message, error)
	CountByConversationID(ctx context.Context, convID uuid.UUID) (int64, error)
}
