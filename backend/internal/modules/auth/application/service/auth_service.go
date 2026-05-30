package service

import (
	"context"
	"time"

	"github.com/jarvas/backend/internal/modules/auth/application/dto"
	"github.com/jarvas/backend/internal/modules/auth/application/port"
	"github.com/jarvas/backend/internal/modules/auth/domain/entity"
	"github.com/jarvas/backend/internal/modules/auth/domain/event"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
	"github.com/jarvas/backend/internal/shared/eventbus"
	"golang.org/x/crypto/bcrypt"
)

// AuthService orchestrates registration, login, and token lifecycle.
// It depends only on interfaces, making it fully testable without I/O.
type AuthService struct {
	userRepo    port.UserRepository
	tokenRepo   port.TokenRepository
	tokenSvc    *TokenService
	bus         *eventbus.Bus
}

func NewAuthService(
	userRepo port.UserRepository,
	tokenRepo port.TokenRepository,
	tokenSvc *TokenService,
	bus *eventbus.Bus,
) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		tokenSvc:  tokenSvc,
		bus:       bus,
	}
}

func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest, ip, ua string) (*dto.AuthResponse, error) {
	existing, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		// 404 = user not found = expected path; anything else is a real DB error.
		if appErr, ok := apperrors.As(err); !ok || appErr.HTTPStatus != 404 {
			return nil, apperrors.Internal(err)
		}
	}
	if existing != nil {
		return nil, apperrors.Conflict("email already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	user := entity.NewUser(req.Email, req.FullName, string(hash))
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, apperrors.Internal(err)
	}

	s.bus.Publish(ctx, event.UserRegistered{
		UserID:    user.ID,
		Email:     user.Email,
		FullName:  user.FullName,
		Provider:  string(user.Provider),
		OccuredAt: time.Now().UTC(),
	})

	return s.buildAuthResponse(ctx, user, ip, ua)
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest, ip, ua string) (*dto.AuthResponse, error) {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil || user == nil {
		return nil, apperrors.Unauthorized("invalid credentials")
	}

	if !user.CanLogin() {
		return nil, apperrors.Forbidden("account is not active")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, apperrors.Unauthorized("invalid credentials")
	}

	_ = s.userRepo.UpdateLastLogin(ctx, user.ID)

	s.bus.Publish(ctx, event.UserLoggedIn{
		UserID:    user.ID,
		IPAddress: ip,
		OccuredAt: time.Now().UTC(),
	})

	return s.buildAuthResponse(ctx, user, ip, ua)
}

func (s *AuthService) RefreshTokens(ctx context.Context, rawToken, ip, ua string) (*dto.TokenPair, error) {
	hash := s.tokenSvc.HashToken(rawToken)
	stored, err := s.tokenRepo.FindByHash(ctx, hash)
	if err != nil || stored == nil {
		return nil, apperrors.Unauthorized("refresh token not found")
	}

	if !stored.IsValid() {
		return nil, apperrors.Unauthorized("refresh token expired or revoked")
	}

	user, err := s.userRepo.FindByID(ctx, stored.UserID)
	if err != nil || user == nil {
		return nil, apperrors.Unauthorized("user not found")
	}

	// Rotate: revoke old, issue new.
	if err := s.tokenRepo.RevokeByHash(ctx, hash); err != nil {
		return nil, apperrors.Internal(err)
	}

	return s.issueTokens(ctx, user, ip, ua)
}

func (s *AuthService) Logout(ctx context.Context, rawToken string, userID string) error {
	hash := s.tokenSvc.HashToken(rawToken)
	return s.tokenRepo.RevokeByHash(ctx, hash)
}

func (s *AuthService) HandleOAuthUser(ctx context.Context, email, fullName, providerID string, provider entity.AuthProvider, ip, ua string) (*dto.AuthResponse, error) {
	user, _ := s.userRepo.FindByProviderID(ctx, string(provider), providerID)

	if user == nil {
		// Check if email already exists (user switching to OAuth).
		user, _ = s.userRepo.FindByEmail(ctx, email)
		if user == nil {
			user = entity.NewOAuthUser(email, fullName, providerID, provider)
			if err := s.userRepo.Create(ctx, user); err != nil {
				return nil, apperrors.Internal(err)
			}
			s.bus.Publish(ctx, event.UserRegistered{
				UserID:    user.ID,
				Email:     user.Email,
				FullName:  user.FullName,
				Provider:  string(provider),
				OccuredAt: time.Now().UTC(),
			})
		} else {
			// Link OAuth to existing account.
			user.Provider = provider
			user.ProviderID = providerID
			user.EmailVerified = true
			if err := s.userRepo.Update(ctx, user); err != nil {
				return nil, apperrors.Internal(err)
			}
		}
	}

	if !user.CanLogin() {
		return nil, apperrors.Forbidden("account is not active")
	}

	_ = s.userRepo.UpdateLastLogin(ctx, user.ID)

	return s.buildAuthResponse(ctx, user, ip, ua)
}

func (s *AuthService) buildAuthResponse(ctx context.Context, user *entity.User, ip, ua string) (*dto.AuthResponse, error) {
	pair, err := s.issueTokens(ctx, user, ip, ua)
	if err != nil {
		return nil, err
	}
	return &dto.AuthResponse{
		TokenPair: *pair,
		User:      toUserResponse(user),
	}, nil
}

func (s *AuthService) issueTokens(ctx context.Context, user *entity.User, ip, ua string) (*dto.TokenPair, error) {
	accessToken, err := s.tokenSvc.GenerateAccessToken(user)
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	raw, hash, err := s.tokenSvc.GenerateRefreshToken()
	if err != nil {
		return nil, apperrors.Internal(err)
	}

	rt := entity.NewRefreshToken(user.ID, hash, ip, ua, s.tokenSvc.RefreshExpiry())
	if err := s.tokenRepo.Save(ctx, rt); err != nil {
		return nil, apperrors.Internal(err)
	}

	return &dto.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: raw,
		ExpiresIn:    s.tokenSvc.AccessExpirySeconds(),
		TokenType:    "Bearer",
	}, nil
}

func toUserResponse(u *entity.User) dto.UserResponse {
	return dto.UserResponse{
		ID:            u.ID.String(),
		Email:         u.Email,
		FullName:      u.FullName,
		AvatarURL:     u.AvatarURL,
		Role:          string(u.Role),
		EmailVerified: u.EmailVerified,
	}
}
