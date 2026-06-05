package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/voice/application/service"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
	"github.com/jarvas/backend/internal/shared/response"
)

type VoiceHandler struct {
	svc *service.VoiceService
}

func NewVoiceHandler(svc *service.VoiceService) *VoiceHandler {
	return &VoiceHandler{svc: svc}
}

// Upload handles multipart audio upload.
// Form fields: audio (file), conversation_id (string), language (optional string).
func (h *VoiceHandler) Upload(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))

	if err := c.Request.ParseMultipartForm(26 << 20); err != nil { // 26 MB max
		response.Error(c, apperrors.BadRequest("failed to parse form"))
		return
	}

	conversationID := c.PostForm("conversation_id")
	if conversationID == "" {
		response.Error(c, apperrors.BadRequest("conversation_id is required"))
		return
	}
	language := c.PostForm("language") // optional

	file, header, err := c.Request.FormFile("audio")
	if err != nil {
		response.Error(c, apperrors.BadRequest("audio file is required"))
		return
	}
	defer file.Close()

	resp, err := h.svc.Upload(c.Request.Context(), userID, conversationID, language, file, header)
	if err != nil {
		response.Error(c, err)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"success": true,
		"data":    resp,
	})
}

// GetSession returns the status and transcript of a voice session.
func (h *VoiceHandler) GetSession(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.BadRequest("invalid session id"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	session, err := h.svc.GetSession(c.Request.Context(), sessionID, userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, session)
}

// ListSessions returns the user's recent voice sessions.
func (h *VoiceHandler) ListSessions(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))

	sessions, err := h.svc.ListSessions(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, sessions)
}
