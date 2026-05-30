package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/auth/domain/entity"
	"github.com/jarvas/backend/internal/shared/config"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
)

// Claims is the JWT payload. Sub is always the user UUID string.
type Claims struct {
	UserID string          `json:"sub"`
	Email  string          `json:"email"`
	Role   entity.UserRole `json:"role"`
	jwt.RegisteredClaims
}

// TokenService handles JWT generation, validation, and refresh token hashing.
// It has no I/O — it is a pure value object from a DI perspective.
type TokenService struct {
	secret        []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewTokenService(cfg config.JWTConfig) *TokenService {
	return &TokenService{
		secret:        []byte(cfg.Secret),
		accessExpiry:  cfg.AccessExpiry,
		refreshExpiry: cfg.RefreshExpiry,
	}
}

func (s *TokenService) GenerateAccessToken(user *entity.User) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		UserID: user.ID.String(),
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessExpiry)),
			Issuer:    "jarvas",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *TokenService) ValidateAccessToken(raw string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(raw, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, apperrors.Unauthorized("invalid or expired token", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, apperrors.Unauthorized("invalid token claims")
	}
	return claims, nil
}

// GenerateRefreshToken creates a cryptographically random opaque token.
// Returns (rawToken, hashedToken, error). Store only the hash in the DB.
func (s *TokenService) GenerateRefreshToken() (raw, hashed string, err error) {
	raw = uuid.New().String() + uuid.New().String() // 72 chars of entropy
	hashed = s.HashToken(raw)
	return raw, hashed, nil
}

func (s *TokenService) HashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func (s *TokenService) RefreshExpiry() time.Duration {
	return s.refreshExpiry
}

func (s *TokenService) AccessExpirySeconds() int {
	return int(s.accessExpiry.Seconds())
}
