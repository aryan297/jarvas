package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/tenant/application/dto"
	"github.com/jarvas/backend/internal/modules/tenant/application/port"
	"github.com/jarvas/backend/internal/modules/tenant/domain/entity"
	tenantrepo "github.com/jarvas/backend/internal/modules/tenant/infrastructure/repository"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
)

type TenantService struct {
	tenantRepo port.TenantRepository
	memberRepo port.MemberRepository
}

func NewTenantService(tenantRepo port.TenantRepository, memberRepo port.MemberRepository) *TenantService {
	return &TenantService{tenantRepo: tenantRepo, memberRepo: memberRepo}
}

// CreatePersonalTenant is called when a user registers — gives them a default workspace.
func (s *TenantService) CreatePersonalTenant(ctx context.Context, userID uuid.UUID, displayName string) (*entity.Tenant, error) {
	t := entity.NewTenant(userID, displayName+"'s Workspace", "")
	t.Slug = tenantrepo.Slugify(t.Name, t.ID)
	if err := s.tenantRepo.Save(ctx, t); err != nil {
		return nil, apperrors.Internal(err)
	}
	owner := entity.NewMember(t.ID, userID, entity.RoleOwner, nil)
	_ = s.memberRepo.AddMember(ctx, owner)
	return t, nil
}

func (s *TenantService) Create(ctx context.Context, userID uuid.UUID, req dto.CreateTenantRequest) (*dto.TenantResponse, error) {
	t := entity.NewTenant(userID, req.Name, "")
	t.Slug = tenantrepo.Slugify(t.Name, t.ID)
	if err := s.tenantRepo.Save(ctx, t); err != nil {
		return nil, apperrors.Internal(err)
	}
	owner := entity.NewMember(t.ID, userID, entity.RoleOwner, nil)
	if err := s.memberRepo.AddMember(ctx, owner); err != nil {
		return nil, apperrors.Internal(err)
	}
	return toDTO(t), nil
}

// ListForUser returns all tenants the user belongs to.
func (s *TenantService) ListForUser(ctx context.Context, userID uuid.UUID) ([]*dto.TenantResponse, error) {
	tenants, err := s.memberRepo.ListTenantsByUser(ctx, userID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	result := make([]*dto.TenantResponse, 0, len(tenants))
	for _, t := range tenants {
		result = append(result, toDTO(t))
	}
	return result, nil
}

func (s *TenantService) GetByID(ctx context.Context, tenantID, requesterID uuid.UUID) (*dto.TenantResponse, error) {
	// Verify the requester is a member.
	if _, err := s.memberRepo.FindMember(ctx, tenantID, requesterID); err != nil {
		return nil, apperrors.Forbidden("not a member of this tenant")
	}
	t, err := s.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return toDTO(t), nil
}

// InviteMember looks up a user by email and adds them to the tenant.
func (s *TenantService) InviteMember(ctx context.Context, tenantID, inviterID uuid.UUID, req dto.InviteMemberRequest) error {
	// Only owner/admin can invite.
	inviter, err := s.memberRepo.FindMember(ctx, tenantID, inviterID)
	if err != nil {
		return apperrors.Forbidden("not a member of this tenant")
	}
	if inviter.Role == entity.RoleMember {
		return apperrors.Forbidden("only admins and owners can invite members")
	}

	targetID, err := s.memberRepo.FindUserByEmail(ctx, req.Email)
	if err != nil {
		return err // surfaces "user not found" to client
	}

	role := entity.RoleMember
	if req.Role == "ADMIN" {
		role = entity.RoleAdmin
	}

	m := entity.NewMember(tenantID, targetID, role, &inviterID)
	if err := s.memberRepo.AddMember(ctx, m); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

// ListMembers returns all members of a tenant.
func (s *TenantService) ListMembers(ctx context.Context, tenantID, requesterID uuid.UUID) ([]*dto.MemberResponse, error) {
	if _, err := s.memberRepo.FindMember(ctx, tenantID, requesterID); err != nil {
		return nil, apperrors.Forbidden("not a member of this tenant")
	}
	members, err := s.memberRepo.ListMembers(ctx, tenantID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	result := make([]*dto.MemberResponse, 0, len(members))
	for _, m := range members {
		result = append(result, toMemberDTO(m))
	}
	return result, nil
}

// RemoveMember removes a user from a tenant. Owners cannot be removed.
func (s *TenantService) RemoveMember(ctx context.Context, tenantID, memberUserID, requesterID uuid.UUID) error {
	requester, err := s.memberRepo.FindMember(ctx, tenantID, requesterID)
	if err != nil {
		return apperrors.Forbidden("not a member of this tenant")
	}
	if requester.Role == entity.RoleMember && requesterID != memberUserID {
		return apperrors.Forbidden("only admins and owners can remove members")
	}
	target, err := s.memberRepo.FindMember(ctx, tenantID, memberUserID)
	if err != nil {
		return err
	}
	if target.Role == entity.RoleOwner {
		return apperrors.BadRequest("tenant owner cannot be removed")
	}
	return s.memberRepo.RemoveMember(ctx, tenantID, memberUserID)
}

// ── DTO mappers ───────────────────────────────────────────────────────────────

func toDTO(t *entity.Tenant) *dto.TenantResponse {
	return &dto.TenantResponse{
		ID:        t.ID.String(),
		Name:      t.Name,
		Slug:      t.Slug,
		Plan:      string(t.Plan),
		OwnerID:   t.OwnerID.String(),
		IsActive:  t.IsActive,
		CreatedAt: t.CreatedAt.Format(time.RFC3339),
	}
}

func toMemberDTO(m *entity.TenantMember) *dto.MemberResponse {
	return &dto.MemberResponse{
		ID:           m.ID.String(),
		UserID:       m.UserID.String(),
		UserEmail:    m.UserEmail,
		UserFullName: m.UserFullName,
		Role:         string(m.Role),
		JoinedAt:     m.JoinedAt.Format(time.RFC3339),
	}
}
