package http

import (
	"github.com/gin-gonic/gin"
	"github.com/jarvas/backend/internal/shared/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, h *RAGHandler, mw *middleware.AuthMiddleware) {
	g := rg.Group("/rag", mw.Authenticate())
	{
		g.POST("/search", h.Search)
	}
}
