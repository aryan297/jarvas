package http

import (
	"github.com/gin-gonic/gin"
	"github.com/jarvas/backend/internal/shared/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, h *DocumentHandler, mw *middleware.AuthMiddleware) {
	g := rg.Group("/documents", mw.Authenticate())
	{
		g.POST("", h.Upload)
		g.GET("", h.List)
		g.GET("/:id", h.GetByID)
		g.DELETE("/:id", h.Delete)
		g.GET("/:id/url", h.GetDownloadURL)
		g.GET("/:id/chunks", h.ListChunks)
	}
}
