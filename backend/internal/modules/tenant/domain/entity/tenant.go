package entity

import (
	"time"

	"github.com/google/uuid"
)

type TenantRole string

const (
	RoleOwner  TenantRole = "OWNER"
	RoleAdmin  TenantRole = "ADMIN"
	RoleMember TenantRole = "MEMBER"
)

type TenantPlan string

const (
	PlanFree       TenantPlan = "free"
	PlanPro        TenantPlan = "pro"
	PlanEnterprise TenantPlan = "enterprise"
)

type Tenant struct {
	ID        uuid.UUID
	Name      string
	Slug      string
	Plan      TenantPlan
	OwnerID   uuid.UUID
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TenantMember struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	UserID    uuid.UUID
	Role      TenantRole
	InvitedBy *uuid.UUID
	JoinedAt  time.Time

	// Populated via JOIN in list queries.
	UserEmail    string
	UserFullName string
}

func NewTenant(ownerID uuid.UUID, name, slug string) *Tenant {
	now := time.Now().UTC()
	return &Tenant{
		ID:        uuid.New(),
		Name:      name,
		Slug:      slug,
		Plan:      PlanFree,
		OwnerID:   ownerID,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func NewMember(tenantID, userID uuid.UUID, role TenantRole, invitedBy *uuid.UUID) *TenantMember {
	return &TenantMember{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    userID,
		Role:      role,
		InvitedBy: invitedBy,
		JoinedAt:  time.Now().UTC(),
	}
}
