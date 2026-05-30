package validator

import (
	"github.com/go-playground/validator/v10"
	apperrors "github.com/jarvas/backend/internal/shared/errors"
)

var validate = validator.New()

// Validate checks a struct against its `validate` tags and returns an
// AppError with a human-readable message on failure.
func Validate(s interface{}) error {
	if err := validate.Struct(s); err != nil {
		errs, ok := err.(validator.ValidationErrors)
		if !ok {
			return apperrors.BadRequest("invalid request body")
		}
		// Return only the first validation error to keep responses concise.
		e := errs[0]
		return apperrors.BadRequest(buildMessage(e))
	}
	return nil
}

func buildMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return e.Field() + " is required"
	case "email":
		return e.Field() + " must be a valid email"
	case "min":
		return e.Field() + " must be at least " + e.Param() + " characters"
	case "max":
		return e.Field() + " must be at most " + e.Param() + " characters"
	case "oneof":
		return e.Field() + " must be one of: " + e.Param()
	default:
		return e.Field() + " is invalid"
	}
}
