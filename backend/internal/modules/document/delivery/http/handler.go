package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jarvas/backend/internal/modules/document/application/service"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
	"github.com/jarvas/backend/internal/shared/response"
)

type DocumentHandler struct {
	svc *service.DocumentService
}

func NewDocumentHandler(svc *service.DocumentService) *DocumentHandler {
	return &DocumentHandler{svc: svc}
}

// Upload godoc — POST /documents  (multipart/form-data, field: file)
func (h *DocumentHandler) Upload(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.Error(c, apperrors.BadRequest("file field is required"))
		return
	}
	defer file.Close()

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	doc, err := h.svc.Upload(c.Request.Context(), userID, header.Filename, mimeType, header.Size, file)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, doc)
}

// List godoc — GET /documents?page=1&limit=20
func (h *DocumentHandler) List(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))
	page, limit := pageLimit(c)

	docs, total, err := h.svc.List(c.Request.Context(), userID, limit, (page-1)*limit)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Paginated(c, docs, response.PaginationMeta{
		Page: page, Limit: limit, Total: total,
		TotalPages: int((total + int64(limit) - 1) / int64(limit)),
	})
}

// GetByID godoc — GET /documents/:id
func (h *DocumentHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.BadRequest("invalid document id"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	doc, err := h.svc.GetByID(c.Request.Context(), id, userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, doc)
}

// Delete godoc — DELETE /documents/:id
func (h *DocumentHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.BadRequest("invalid document id"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	if err := h.svc.Delete(c.Request.Context(), id, userID); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContent(c)
}

// GetDownloadURL godoc — GET /documents/:id/url
func (h *DocumentHandler) GetDownloadURL(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.BadRequest("invalid document id"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	url, err := h.svc.GetPresignedURL(c.Request.Context(), id, userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, gin.H{"url": url})
}

// ListChunks godoc — GET /documents/:id/chunks
func (h *DocumentHandler) ListChunks(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, apperrors.BadRequest("invalid document id"))
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	chunks, err := h.svc.ListChunks(c.Request.Context(), id, userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, chunks)
}

func pageLimit(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return page, limit
}
