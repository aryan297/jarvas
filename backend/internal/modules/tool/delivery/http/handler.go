package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/tool/application/port"
	toolentity "github.com/jarvas/backend/internal/modules/tool/domain/entity"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
	"github.com/jarvas/backend/internal/shared/response"
)

type ToolHandler struct {
	toolRepo       port.ToolRepository
	configRepo     port.UserToolConfigRepository
}

func NewToolHandler(toolRepo port.ToolRepository, configRepo port.UserToolConfigRepository) *ToolHandler {
	return &ToolHandler{toolRepo: toolRepo, configRepo: configRepo}
}

// ListTools returns all registered tools.
func (h *ToolHandler) ListTools(c *gin.Context) {
	tools, err := h.toolRepo.FindAll(c.Request.Context())
	if err != nil {
		response.Error(c, apperrors.Internal(err))
		return
	}

	type toolDTO struct {
		ID          string      `json:"id"`
		Name        string      `json:"name"`
		DisplayName string      `json:"display_name"`
		Description string      `json:"description"`
		Category    string      `json:"category"`
		IsBuiltin   bool        `json:"is_builtin"`
		Schema      interface{} `json:"schema"`
	}
	result := make([]toolDTO, 0, len(tools))
	for _, t := range tools {
		result = append(result, toolDTO{
			ID:          t.ID.String(),
			Name:        t.Name,
			DisplayName: t.DisplayName,
			Description: t.Description,
			Category:    string(t.Category),
			IsBuiltin:   t.IsBuiltin,
			Schema:      t.Schema,
		})
	}
	response.OK(c, result)
}

// ConfigureTool saves or updates user credentials for a tool.
func (h *ToolHandler) ConfigureTool(c *gin.Context) {
	toolName := c.Param("name")
	userID, _ := uuid.Parse(c.GetString("user_id"))

	tool, err := h.toolRepo.FindByName(c.Request.Context(), toolName)
	if err != nil {
		response.Error(c, err)
		return
	}

	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, apperrors.BadRequest("invalid config body"))
		return
	}

	now := time.Now().UTC()
	cfg := &toolentity.UserToolConfig{
		ID:        uuid.New(),
		UserID:    userID,
		ToolID:    tool.ID,
		Config:    body,
		IsEnabled: true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := h.configRepo.Save(c.Request.Context(), cfg); err != nil {
		response.Error(c, apperrors.Internal(err))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "tool configured"})
}

// GetToolConfig returns the user's saved config for a tool.
func (h *ToolHandler) GetToolConfig(c *gin.Context) {
	toolName := c.Param("name")
	userID, _ := uuid.Parse(c.GetString("user_id"))

	tool, err := h.toolRepo.FindByName(c.Request.Context(), toolName)
	if err != nil {
		response.Error(c, err)
		return
	}

	cfg, err := h.configRepo.FindByUserAndTool(c.Request.Context(), userID, tool.ID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{
		"tool_id":    cfg.ToolID.String(),
		"is_enabled": cfg.IsEnabled,
		"config":     cfg.Config,
	})
}
