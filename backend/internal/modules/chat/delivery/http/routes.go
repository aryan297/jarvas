package http

import (
	"github.com/gin-gonic/gin"
	"github.com/jarvas/backend/internal/shared/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, h *ChatHandler, mw *middleware.AuthMiddleware) {
	g := rg.Group("/conversations", mw.Authenticate())
	{
		g.POST("", h.CreateConversation)
		g.GET("", h.ListConversations)
		g.GET("/:id", h.GetConversation)
		g.DELETE("/:id", h.DeleteConversation)
		g.POST("/:id/messages", h.SendMessage)
		g.GET("/:id/messages", h.ListMessages)
	}
}
