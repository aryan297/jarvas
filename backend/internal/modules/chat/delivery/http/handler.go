package http

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/chat/application/dto"
	"github.com/jarvas/backend/internal/modules/chat/application/service"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
	"github.com/jarvas/backend/internal/shared/response"
	"github.com/jarvas/backend/internal/shared/validator"
)

type ChatHandler struct {
	chatSvc *service.ChatService
}

func NewChatHandler(svc *service.ChatService) *ChatHandler {
	return &ChatHandler{chatSvc: svc}
}

// CreateConversation godoc
// POST /conversations
func (h *ChatHandler) CreateConversation(c *gin.Context) {
	var req dto.CreateConversationRequest
	_ = c.ShouldBindJSON(&req)
	userID, _ := uuid.Parse(c.GetString("user_id"))

	conv, err := h.chatSvc.CreateConversation(c.Request.Context(), userID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, conv)
}

// ListConversations godoc
// GET /conversations?page=1&limit=20
func (h *ChatHandler) ListConversations(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))
	page, limit := pageLimit(c)

	convs, total, err := h.chatSvc.ListConversations(c.Request.Context(), userID, limit, (page-1)*limit)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Paginated(c, convs, response.PaginationMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages(total, limit),
	})
}

// GetConversation godoc
// GET /conversations/:id
func (h *ChatHandler) GetConversation(c *gin.Context) {
	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.BadRequest("invalid conversation id"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	conv, err := h.chatSvc.GetConversation(c.Request.Context(), convID, userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, conv)
}

// DeleteConversation godoc
// DELETE /conversations/:id
func (h *ChatHandler) DeleteConversation(c *gin.Context) {
	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.BadRequest("invalid conversation id"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	if err := h.chatSvc.ArchiveConversation(c.Request.Context(), convID, userID); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}

// SendMessage godoc
// POST /conversations/:id/messages
// Supports SSE streaming when body.stream=true
func (h *ChatHandler) SendMessage(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))

	var req dto.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.BadRequest("invalid request body"))
		return
	}
	if err := validator.Validate(&req); err != nil {
		response.Error(c, err)
		return
	}
	req.ConversationID = c.Param("id")

	// ── SSE streaming path ────────────────────────────────────────────────────
	if req.Stream {
		ch, err := h.chatSvc.StreamMessage(c.Request.Context(), userID, req)
		if err != nil {
			response.Error(c, err)
			return
		}

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no") // disable nginx buffering

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			response.Error(c, apperrors.Internal(nil))
			return
		}

		for delta := range ch {
			fmt.Fprintf(c.Writer, "data: %s\n\n", delta)
			flusher.Flush()
		}
		fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
		flusher.Flush()
		return
	}

	// ── Non-streaming path ────────────────────────────────────────────────────
	msg, err := h.chatSvc.SendMessage(c.Request.Context(), userID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, msg)
}

// ListMessages godoc
// GET /conversations/:id/messages?page=1&limit=50
func (h *ChatHandler) ListMessages(c *gin.Context) {
	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.BadRequest("invalid conversation id"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))
	page, limit := pageLimit(c)

	msgs, total, err := h.chatSvc.ListMessages(c.Request.Context(), convID, userID, limit, (page-1)*limit)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Paginated(c, msgs, response.PaginationMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages(total, limit),
	})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func pageLimit(c *gin.Context) (page, limit int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ = strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return
}

func totalPages(total int64, limit int) int {
	if limit == 0 {
		return 0
	}
	return int((total + int64(limit) - 1) / int64(limit))
}
