package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jarvas/backend/internal/modules/auth/application/service"
	"github.com/jarvas/backend/internal/modules/auth/domain/entity"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
	"github.com/jarvas/backend/internal/shared/response"
)

// AuthMiddleware validates JWT tokens and injects claims into the Gin context.
type AuthMiddleware struct {
	tokenSvc *service.TokenService
}

func NewAuthMiddleware(tokenSvc *service.TokenService) *AuthMiddleware {
	return &AuthMiddleware{tokenSvc: tokenSvc}
}

// Authenticate validates the Bearer token and populates context values:
//   - "user_id": string UUID
//   - "user_email": string
//   - "user_role": entity.UserRole
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := extractBearerToken(c)
		if raw == "" {
			response.Error(c, apperrors.Unauthorized("authorization header missing"))
			c.Abort()
			return
		}

		claims, err := m.tokenSvc.ValidateAccessToken(raw)
		if err != nil {
			response.Error(c, err)
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Next()
	}
}

// RequireRole returns a middleware that enforces the given role(s).
// Must be used after Authenticate().
func (m *AuthMiddleware) RequireRole(roles ...entity.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("user_role")
		if !exists {
			response.Error(c, apperrors.Unauthorized("not authenticated"))
			c.Abort()
			return
		}

		userRole, ok := roleVal.(entity.UserRole)
		if !ok {
			response.Error(c, apperrors.Internal(nil))
			c.Abort()
			return
		}

		for _, r := range roles {
			if userRole == r {
				c.Next()
				return
			}
		}

		response.Error(c, apperrors.Forbidden("insufficient permissions"))
		c.Abort()
	}
}

func extractBearerToken(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return parts[1]
}
