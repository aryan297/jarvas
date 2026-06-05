package http

import (
	"github.com/gin-gonic/gin"
	"github.com/jarvas/backend/internal/shared/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, h *VoiceHandler, mw *middleware.AuthMiddleware) {
	g := rg.Group("/voice", mw.Authenticate())
	{
		g.POST("/upload",      h.Upload)
		g.GET("/sessions",     h.ListSessions)
		g.GET("/sessions/:id", h.GetSession)
	}
}
