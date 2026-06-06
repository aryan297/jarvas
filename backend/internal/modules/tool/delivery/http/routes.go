package http

import (
	"github.com/gin-gonic/gin"
	"github.com/jarvas/backend/internal/shared/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, h *ToolHandler, mw *middleware.AuthMiddleware) {
	g := rg.Group("/tools", mw.Authenticate())
	{
		g.GET("",                   h.ListTools)
		g.GET("/:name/config",      h.GetToolConfig)
		g.POST("/:name/configure",  h.ConfigureTool)
	}
}
