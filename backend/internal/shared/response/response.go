package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
	"github.com/jarvas/backend/internal/shared/logger"
	"go.uber.org/zap"
)

type successBody struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

type errorBody struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, successBody{Success: true, Data: data})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, successBody{Success: true, Data: data})
}

func Paginated(c *gin.Context, data interface{}, meta PaginationMeta) {
	c.JSON(http.StatusOK, successBody{Success: true, Data: data, Meta: meta})
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Error writes the HTTP error response and logs 5xx causes for debugging.
func Error(c *gin.Context, err error) {
	if appErr, ok := apperrors.As(err); ok {
		if appErr.HTTPStatus >= 500 && appErr.Err != nil {
			logger.Error("internal error",
				zap.String("path", c.Request.URL.Path),
				zap.Error(appErr.Err),
			)
		}
		c.JSON(appErr.HTTPStatus, errorBody{
			Success: false,
			Code:    appErr.Code,
			Message: appErr.Message,
		})
		return
	}
	logger.Error("unhandled error", zap.String("path", c.Request.URL.Path), zap.Error(err))
	c.JSON(http.StatusInternalServerError, errorBody{
		Success: false,
		Code:    "INTERNAL_ERROR",
		Message: "an unexpected error occurred",
	})
}
