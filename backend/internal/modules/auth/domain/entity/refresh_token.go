package entity

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	DeviceID  string
	IPAddress string
	UserAgent string
	Revoked   bool
	ExpiresAt time.Time
	CreatedAt time.Time
}

func NewRefreshToken(userID uuid.UUID, tokenHash, ip, userAgent string, ttl time.Duration) *RefreshToken {
	return &RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: tokenHash,
		IPAddress: ip,
		UserAgent: userAgent,
		ExpiresAt: time.Now().UTC().Add(ttl),
		CreatedAt: time.Now().UTC(),
	}
}

func (t *RefreshToken) IsValid() bool {
	return !t.Revoked && time.Now().UTC().Before(t.ExpiresAt)
}
