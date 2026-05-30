package http

import (
	"github.com/gin-gonic/gin"
	"github.com/jarvas/backend/internal/shared/middleware"
)

// RegisterRoutes mounts all auth routes onto the provided router group.
func RegisterRoutes(rg *gin.RouterGroup, h *AuthHandler, mw *middleware.AuthMiddleware) {
	auth := rg.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.Refresh)
		auth.GET("/google/login", h.GoogleLogin)
		auth.GET("/google/callback", h.GoogleCallback)

		// Protected routes
		protected := auth.Group("", mw.Authenticate())
		{
			protected.POST("/logout", h.Logout)
			protected.GET("/me", h.Me)
		}
	}
}
