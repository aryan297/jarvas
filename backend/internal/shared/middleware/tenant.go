package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
	"github.com/jarvas/backend/internal/shared/response"
)

// TenantMemberChecker validates that a user belongs to the requested tenant.
// Implemented by tenant.MemberRepository to avoid an import cycle.
type TenantMemberChecker interface {
	FindMember(ctx interface{ Value(interface{}) interface{} }, tenantID, userID uuid.UUID) (interface{}, error)
}

// TenantLookup is the minimal interface the middleware uses — avoids importing the tenant package.
type TenantLookup interface {
	IsMember(tenantID, userID uuid.UUID) bool
}

// ValidateTenant reads the X-Tenant-ID header and verifies the current user is a member.
// Must be used after Authenticate().
// If X-Tenant-ID is absent the request passes through (user-scoped operation).
func ValidateTenant(checker func(tenantID, userID uuid.UUID) bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawTenantID := c.GetHeader("X-Tenant-ID")
		if rawTenantID == "" {
			c.Next()
			return
		}

		tenantID, err := uuid.Parse(rawTenantID)
		if err != nil {
			response.Error(c, apperrors.BadRequest("invalid X-Tenant-ID header"))
			c.Abort()
			return
		}

		userID, _ := uuid.Parse(c.GetString("user_id"))
		if !checker(tenantID, userID) {
			response.Error(c, apperrors.Forbidden("not a member of this tenant"))
			c.Abort()
			return
		}

		c.Set("tenant_id", tenantID.String())
		c.Next()
	}
}
