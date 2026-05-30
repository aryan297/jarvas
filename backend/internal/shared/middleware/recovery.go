package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jarvas/backend/internal/shared/logger"
	"go.uber.org/zap"
)

// Recovery catches panics, logs them with a stack trace, and returns a 500.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered",
					zap.Any("panic", r),
					zap.String("path", c.Request.URL.Path),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"code":    "INTERNAL_ERROR",
					"message": "an unexpected error occurred",
				})
			}
		}()
		c.Next()
	}
}
