package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/auth/domain/entity"
)

// UserRepository defines the persistence contract for User aggregates.
// Implementations live in infrastructure/repository.
type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	FindByProviderID(ctx context.Context, provider, providerID string) (*entity.User, error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
	Update(ctx context.Context, user *entity.User) error
}

// TokenRepository manages refresh token persistence.
type TokenRepository interface {
	Save(ctx context.Context, token *entity.RefreshToken) error
	FindByHash(ctx context.Context, hash string) (*entity.RefreshToken, error)
	RevokeByHash(ctx context.Context, hash string) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) error
}
