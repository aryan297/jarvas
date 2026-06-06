package http

import (
	"github.com/gin-gonic/gin"
	"github.com/jarvas/backend/internal/shared/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, h *TenantHandler, mw *middleware.AuthMiddleware) {
	g := rg.Group("/tenants", mw.Authenticate())
	{
		g.POST("",                         h.CreateTenant)
		g.GET("",                          h.ListMyTenants)
		g.GET("/:id",                      h.GetTenant)
		g.POST("/:id/invite",              h.InviteMember)
		g.GET("/:id/members",              h.ListMembers)
		g.DELETE("/:id/members/:user_id",  h.RemoveMember)
	}
}
