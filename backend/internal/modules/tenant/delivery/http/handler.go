package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/tenant/application/dto"
	"github.com/jarvas/backend/internal/modules/tenant/application/service"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
	"github.com/jarvas/backend/internal/shared/response"
)

type TenantHandler struct {
	svc *service.TenantService
}

func NewTenantHandler(svc *service.TenantService) *TenantHandler {
	return &TenantHandler{svc: svc}
}

// CreateTenant creates a new tenant with the caller as owner.
func (h *TenantHandler) CreateTenant(c *gin.Context) {
	var req dto.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.BadRequest("invalid request body"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	t, err := h.svc.Create(c.Request.Context(), userID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, t)
}

// ListMyTenants returns all tenants the caller belongs to.
func (h *TenantHandler) ListMyTenants(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))
	tenants, err := h.svc.ListForUser(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, tenants)
}

// GetTenant returns a tenant the caller is a member of.
func (h *TenantHandler) GetTenant(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.BadRequest("invalid tenant id"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	t, err := h.svc.GetByID(c.Request.Context(), tenantID, userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, t)
}

// InviteMember adds a user to a tenant by email.
func (h *TenantHandler) InviteMember(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.BadRequest("invalid tenant id"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	var req dto.InviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.BadRequest("invalid request body"))
		return
	}

	if err := h.svc.InviteMember(c.Request.Context(), tenantID, userID, req); err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "member added"})
}

// ListMembers returns all members of a tenant.
func (h *TenantHandler) ListMembers(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.BadRequest("invalid tenant id"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	members, err := h.svc.ListMembers(c.Request.Context(), tenantID, userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, members)
}

// RemoveMember removes a user from a tenant.
func (h *TenantHandler) RemoveMember(c *gin.Context) {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.BadRequest("invalid tenant id"))
		return
	}
	memberID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		response.Error(c, apperrors.BadRequest("invalid user id"))
		return
	}
	requesterID, _ := uuid.Parse(c.GetString("user_id"))

	if err := h.svc.RemoveMember(c.Request.Context(), tenantID, memberID, requesterID); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
