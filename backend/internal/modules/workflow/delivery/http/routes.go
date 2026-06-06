package http

import (
	"github.com/gin-gonic/gin"
	"github.com/jarvas/backend/internal/shared/middleware"
)

func RegisterRoutes(rg *gin.RouterGroup, h *WorkflowHandler, mw *middleware.AuthMiddleware) {
	g := rg.Group("/workflows", mw.Authenticate())
	{
		g.POST("",                    h.Create)
		g.GET("",                     h.List)
		g.GET("/:id",                 h.Get)
		g.PATCH("/:id",               h.Update)
		g.DELETE("/:id",              h.Delete)
		g.POST("/:id/run",            h.TriggerRun)
		g.GET("/:id/runs",            h.ListRuns)
		g.GET("/:id/runs/:run_id",    h.GetRun)
	}
}
