package http

import (
	"github.com/gin-gonic/gin"
	"github.com/jarvas/backend/internal/shared/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, h *MemoryHandler, mw *middleware.AuthMiddleware) {
	g := rg.Group("/memories", mw.Authenticate())
	{
		g.POST("",         h.CreateMemory)
		g.GET("",          h.ListMemories)
		g.DELETE("/:id",   h.DeleteMemory)
		g.POST("/search",  h.SearchMemories)
	}
}
