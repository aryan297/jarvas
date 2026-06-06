package port

import (
	"context"

	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/tenant/domain/entity"
)

type TenantRepository interface {
	Save(ctx context.Context, t *entity.Tenant) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Tenant, error)
	FindByOwnerID(ctx context.Context, ownerID uuid.UUID) ([]*entity.Tenant, error)
	Update(ctx context.Context, t *entity.Tenant) error
}

type MemberRepository interface {
	AddMember(ctx context.Context, m *entity.TenantMember) error
	FindMember(ctx context.Context, tenantID, userID uuid.UUID) (*entity.TenantMember, error)
	ListMembers(ctx context.Context, tenantID uuid.UUID) ([]*entity.TenantMember, error)
	ListTenantsByUser(ctx context.Context, userID uuid.UUID) ([]*entity.Tenant, error)
	RemoveMember(ctx context.Context, tenantID, userID uuid.UUID) error
	FindUserByEmail(ctx context.Context, email string) (uuid.UUID, error)
}
