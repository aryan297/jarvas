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

// MemoryRetriever is implemented by the memory module service.
// Defined here to avoid an import cycle between chat and memory packages.
type MemoryRetriever interface {
	SearchRelevant(ctx context.Context, userID uuid.UUID, query string, limit int) ([]string, error)
}

// AgentRunnerPort is implemented by agent.RunnerService.
// Defined here to avoid an import cycle between chat and agent packages.
type AgentRunnerPort interface {
	RunMessage(ctx context.Context, agentID, userID uuid.UUID, userMsg string, history []entity.Message) (string, error)
	StreamMessage(ctx context.Context, agentID, userID uuid.UUID, userMsg string, history []entity.Message) (<-chan string, error)
}
