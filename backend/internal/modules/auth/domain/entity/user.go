package entity

import (
	"time"

	"github.com/google/uuid"
)

type UserRole   string
type UserStatus string
type AuthProvider string

const (
	RoleAdmin       UserRole = "ADMIN"
	RoleUser        UserRole = "USER"
	RolePremiumUser UserRole = "PREMIUM_USER"

	StatusActive   UserStatus = "ACTIVE"
	StatusInactive UserStatus = "INACTIVE"
	StatusBanned   UserStatus = "BANNED"

	ProviderLocal  AuthProvider = "LOCAL"
	ProviderGoogle AuthProvider = "GOOGLE"
)

// User is the aggregate root for the auth bounded context.
type User struct {
	ID             uuid.UUID
	Email          string
	PasswordHash   string
	FullName       string
	AvatarURL      string
	Role           UserRole
	Status         UserStatus
	Provider       AuthProvider
	ProviderID     string
	TenantID       *uuid.UUID
	EmailVerified  bool
	LastLoginAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewUser(email, fullName, passwordHash string) *User {
	now := time.Now().UTC()
	return &User{
		ID:           uuid.New(),
		Email:        email,
		FullName:     fullName,
		PasswordHash: passwordHash,
		Role:         RoleUser,
		Status:       StatusActive,
		Provider:     ProviderLocal,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func NewOAuthUser(email, fullName, providerID string, provider AuthProvider) *User {
	u := NewUser(email, fullName, "")
	u.Provider = provider
	u.ProviderID = providerID
	u.EmailVerified = true
	return u
}

func (u *User) IsActive() bool {
	return u.Status == StatusActive
}

func (u *User) CanLogin() bool {
	return u.IsActive()
}

func (u *User) HasRole(role UserRole) bool {
	return u.Role == role
}

func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}
