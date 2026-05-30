package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError is the standard application error type throughout the system.
// It carries HTTP status, a machine-readable code, and a human message.
type AppError struct {
	HTTPStatus int    `json:"-"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	Err        error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// Constructors for common error categories.

func New(status int, code, message string, cause error) *AppError {
	return &AppError{HTTPStatus: status, Code: code, Message: message, Err: cause}
}

func BadRequest(message string, cause ...error) *AppError {
	var err error
	if len(cause) > 0 {
		err = cause[0]
	}
	return New(http.StatusBadRequest, "BAD_REQUEST", message, err)
}

func Unauthorized(message string, cause ...error) *AppError {
	var err error
	if len(cause) > 0 {
		err = cause[0]
	}
	return New(http.StatusUnauthorized, "UNAUTHORIZED", message, err)
}

func Forbidden(message string, cause ...error) *AppError {
	var err error
	if len(cause) > 0 {
		err = cause[0]
	}
	return New(http.StatusForbidden, "FORBIDDEN", message, err)
}

func NotFound(resource string) *AppError {
	return New(http.StatusNotFound, "NOT_FOUND", fmt.Sprintf("%s not found", resource), nil)
}

func Conflict(message string, cause ...error) *AppError {
	var err error
	if len(cause) > 0 {
		err = cause[0]
	}
	return New(http.StatusConflict, "CONFLICT", message, err)
}

func Internal(cause error) *AppError {
	return New(http.StatusInternalServerError, "INTERNAL_ERROR", "an internal error occurred", cause)
}

func Unprocessable(message string) *AppError {
	return New(http.StatusUnprocessableEntity, "UNPROCESSABLE", message, nil)
}

func TooManyRequests() *AppError {
	return New(http.StatusTooManyRequests, "RATE_LIMITED", "too many requests", nil)
}

// As unwraps the error chain looking for *AppError.
func As(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

// Is checks if an error matches the standard library sentinel.
var Is = errors.Is
