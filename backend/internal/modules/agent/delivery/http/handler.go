package http

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/agent/application/dto"
	"github.com/jarvas/backend/internal/modules/agent/application/service"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
	"github.com/jarvas/backend/internal/shared/response"
)

type AgentHandler struct {
	svc *service.AgentService
}

func NewAgentHandler(svc *service.AgentService) *AgentHandler {
	return &AgentHandler{svc: svc}
}

func (h *AgentHandler) CreateAgent(c *gin.Context) {
	var req dto.CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.BadRequest("invalid request body"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	a, err := h.svc.Create(c.Request.Context(), userID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, a)
}

func (h *AgentHandler) ListAgents(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))
	page, limit := pageLimit(c)

	agents, total, err := h.svc.List(c.Request.Context(), userID, limit, (page-1)*limit)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Paginated(c, agents, response.PaginationMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	})
}

func (h *AgentHandler) GetAgent(c *gin.Context) {
	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.BadRequest("invalid agent id"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	a, err := h.svc.GetByID(c.Request.Context(), agentID, userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, a)
}

func (h *AgentHandler) UpdateAgent(c *gin.Context) {
	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.BadRequest("invalid agent id"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	var req dto.UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.BadRequest("invalid request body"))
		return
	}

	a, err := h.svc.Update(c.Request.Context(), agentID, userID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, a)
}

func (h *AgentHandler) DeleteAgent(c *gin.Context) {
	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.BadRequest("invalid agent id"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	if err := h.svc.Delete(c.Request.Context(), agentID, userID); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func pageLimit(c *gin.Context) (int, int) {
	page, limit := 1, 20
	if p := c.Query("page"); p != "" {
		var v int
		if _, err := fmt.Sscan(p, &v); err == nil && v > 0 {
			page = v
		}
	}
	if l := c.Query("limit"); l != "" {
		var v int
		if _, err := fmt.Sscan(l, &v); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	return page, limit
}
