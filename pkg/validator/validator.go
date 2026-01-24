// Package validator provides validation utilities.
// It wraps the go-playground/validator library with custom error formatting.
package validator

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/hapkiduki/order-go/internal/application/dto"
)

// Validator wraps the validator.Validate instance with custom formatting.
type Validator struct {
	validate *validator.Validate
}

// New creates a new Validator instance.
//
// Returns:
//   - *Validator: The validator instance.
func New() *Validator {
	v := validator.New()

	// use json tag names in error messages
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
	return &Validator{
		validate: v,
	}
}

// Validate validates the given struct and returns formatted validation errors.
//
// Parameters:
//   - s: The struct to validate.
//
// Returns:
//   - []dto.ValidationError: List of validation errors (empty if valid).
func (v *Validator) Validate(s interface{}) []dto.ValidationError {
	err := v.validate.Struct(s)
	if err == nil {
		return nil
	}

	var errors []dto.ValidationError
	for _, err := range err.(validator.ValidationErrors) {
		ve := dto.ValidationError{
			Field:   err.Field(),
			Message: formatValidationError(err),
		}
		errors = append(errors, ve)
	}
	return errors
}

// formatValidationError maps validator tags to human-readable error messages.
func formatValidationError(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Must be a valid email address"
	case "url":
		return "Must be a valid URL"
	case "min":
		return "Must be at least " + err.Param()
	case "max":
		return "Must be at most " + err.Param()
	case "len":
		return "Must be exactly " + err.Param() + " characters"
	case "oneof":
		return "Must be one of: " + err.Param()
	case "uuid":
		return "Must be a valid UUID"
	case "gt":
		return "Must be greater than " + err.Param()
	case "gte":
		return "Must be greater than or equal to " + err.Param()
	case "lt":
		return "Must be less than " + err.Param()
	case "lte":
		return "Must be less than or equal to " + err.Param()
	case "alphanum":
		return "Must contain only alphanumeric characters"
	case "numeric":
		return "Must be a number"
	case "datetime":
		return "Must be a valid datetime"
	default:
		return "Validation failed for " + err.Tag()
	}
}

// ValidateVar validates a single variable against a`tag.
//
// Parameters:
//   - field: The variable to validate.
//   - tag: The validation tag.
//
// Returns:
//   - error: Validation error or nil.
func (v *Validator) ValidateVar(field interface{}, tag string) error {
	return v.validate.Var(field, tag)
}

// RegisterValidation registers a custom validation function.
//
// Parameters:
//   - tag: The validation tag.
//   - fn: The validation function.
//
// Returns:
//   - error: Registration error or nil.
func (v *Validator) RegisterValidation(tag string, fn validator.Func) error {
	return v.validate.RegisterValidation(tag, fn)
}
