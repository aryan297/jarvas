package http

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	ragsvc "github.com/jarvas/backend/internal/modules/rag/application/service"
	ragentity "github.com/jarvas/backend/internal/modules/rag/domain/entity"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
	"github.com/jarvas/backend/internal/shared/response"
)

type RAGHandler struct {
	ragSvc *ragsvc.RAGService
}

func NewRAGHandler(ragSvc *ragsvc.RAGService) *RAGHandler {
	return &RAGHandler{ragSvc: ragSvc}
}

type searchRequest struct {
	Query    string  `json:"query"    binding:"required,min=1"`
	TopK     int     `json:"top_k"`
	MinScore float32 `json:"min_score"`
}

type searchResponse struct {
	Chunks []chunkResult `json:"chunks"`
	Total  int           `json:"total"`
}

type chunkResult struct {
	DocumentID string  `json:"document_id"`
	DocName    string  `json:"doc_name"`
	Content    string  `json:"content"`
	Score      float32 `json:"score"`
}

// Search godoc — POST /rag/search
func (h *RAGHandler) Search(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))

	var req searchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperrors.BadRequest("query is required"))
		return
	}

	topK := req.TopK
	if topK == 0 {
		topK = 5
	}
	minScore := req.MinScore
	if minScore == 0 {
		minScore = 0.3
	}

	ctx, err := h.ragSvc.Search(c.Request.Context(), ragentity.SearchQuery{
		UserID:   userID,
		Query:    req.Query,
		TopK:     topK,
		MinScore: minScore,
	})
	if err != nil {
		response.Error(c, err)
		return
	}

	var results []chunkResult
	for _, ch := range ctx.Chunks {
		results = append(results, chunkResult{
			DocumentID: ch.DocumentID.String(),
			DocName:    ch.DocName,
			Content:    ch.Content,
			Score:      ch.Score,
		})
	}
	response.OK(c, searchResponse{Chunks: results, Total: len(results)})
}
