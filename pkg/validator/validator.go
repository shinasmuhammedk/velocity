package validator

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"strings"
	velocityerrors "velocity/pkg/errors"    
)

var validate *validator.Validate

// Initialize the validator once.
func init() {
	validate = validator.New()
}

// Validate validates a struct using validation tags.
func Validate(v interface{}) error {
	if err := validate.Struct(v); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			var messages []string

			for _, fieldErr := range validationErrors {
				messages = append(messages, formatError(fieldErr))
			}

			return velocityerrors.New(
				velocityerrors.CodeValidation,
				strings.Join(messages, ", "),
			)
		}

		return err
	}

	return nil
}

// formatError converts validator errors into readable messages.
func formatError(err validator.FieldError) string {
	field := err.Field()

	switch err.Tag() {

	case "required":
		return fmt.Sprintf("%s is required", field)

	case "email":
		return fmt.Sprintf("%s must be a valid email", field)

	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, err.Param())

	case "max":
		return fmt.Sprintf("%s must not exceed %s characters", field, err.Param())

	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", field, err.Param())

	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", field, err.Param())

	case "oneof":
		return fmt.Sprintf("%s must be one of [%s]", field, err.Param())

	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}
