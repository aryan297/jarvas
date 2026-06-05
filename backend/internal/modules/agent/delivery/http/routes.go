package http

import (
	"github.com/gin-gonic/gin"
	"github.com/jarvas/backend/internal/shared/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, h *AgentHandler, mw *middleware.AuthMiddleware) {
	g := rg.Group("/agents", mw.Authenticate())
	{
		g.POST("",       h.CreateAgent)
		g.GET("",        h.ListAgents)
		g.GET("/:id",    h.GetAgent)
		g.PATCH("/:id",  h.UpdateAgent)
		g.DELETE("/:id", h.DeleteAgent)
	}
}
