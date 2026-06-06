package http

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/workflow/application/dto"
	"github.com/jarvas/backend/internal/modules/workflow/application/service"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
	"github.com/jarvas/backend/internal/shared/response"
)

type WorkflowHandler struct {
	svc *service.WorkflowService
}

func NewWorkflowHandler(svc *service.WorkflowService) *WorkflowHandler {
	return &WorkflowHandler{svc: svc}
}

func (h *WorkflowHandler) Create(c *gin.Context) {
	var req dto.CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.BadRequest("invalid request body"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	wf, err := h.svc.Create(c.Request.Context(), userID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, wf)
}

func (h *WorkflowHandler) List(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))
	page, limit := pageLimit(c)

	wfs, total, err := h.svc.List(c.Request.Context(), userID, limit, (page-1)*limit)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Paginated(c, wfs, response.PaginationMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	})
}

func (h *WorkflowHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.BadRequest("invalid workflow id"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	wf, err := h.svc.GetByID(c.Request.Context(), id, userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, wf)
}

func (h *WorkflowHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.BadRequest("invalid workflow id"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	var req dto.UpdateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.BadRequest("invalid request body"))
		return
	}

	wf, err := h.svc.Update(c.Request.Context(), id, userID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, wf)
}

func (h *WorkflowHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.BadRequest("invalid workflow id"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	if err := h.svc.Delete(c.Request.Context(), id, userID); err != nil {
		response.Error(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *WorkflowHandler) TriggerRun(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.BadRequest("invalid workflow id"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	var req dto.TriggerRunRequest
	_ = c.ShouldBindJSON(&req) // payload is optional

	run, err := h.svc.TriggerRun(c.Request.Context(), id, userID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": run})
}

func (h *WorkflowHandler) ListRuns(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.BadRequest("invalid workflow id"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))
	page, limit := pageLimit(c)

	runs, total, err := h.svc.ListRuns(c.Request.Context(), id, userID, limit, (page-1)*limit)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Paginated(c, runs, response.PaginationMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	})
}

func (h *WorkflowHandler) GetRun(c *gin.Context) {
	runID, err := uuid.Parse(c.Param("run_id"))
	if err != nil {
		response.Error(c, apperrors.BadRequest("invalid run id"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	run, err := h.svc.GetRun(c.Request.Context(), runID, userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, run)
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
