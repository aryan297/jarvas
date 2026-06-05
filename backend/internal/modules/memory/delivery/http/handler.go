package http

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/memory/application/dto"
	"github.com/jarvas/backend/internal/modules/memory/application/service"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
	"github.com/jarvas/backend/internal/shared/response"
)

type MemoryHandler struct {
	svc *service.MemoryService
}

func NewMemoryHandler(svc *service.MemoryService) *MemoryHandler {
	return &MemoryHandler{svc: svc}
}

func (h *MemoryHandler) CreateMemory(c *gin.Context) {
	var req dto.CreateMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.BadRequest("invalid request body"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	m, err := h.svc.Create(c.Request.Context(), userID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, m)
}

func (h *MemoryHandler) ListMemories(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))
	page, limit := pageLimit(c)

	memories, total, err := h.svc.List(c.Request.Context(), userID, limit, (page-1)*limit)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Paginated(c, memories, response.PaginationMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	})
}

func (h *MemoryHandler) DeleteMemory(c *gin.Context) {
	memID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.BadRequest("invalid memory id"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	if err := h.svc.Delete(c.Request.Context(), memID, userID); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *MemoryHandler) SearchMemories(c *gin.Context) {
	var req dto.SearchMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.BadRequest("invalid request body"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	results, err := h.svc.Search(c.Request.Context(), userID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, results)
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
