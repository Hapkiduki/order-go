// Package dto contains data transfer objects.
package dto

// PaginateResponse represents a paginated list of items.
// It is generic to support any item type.
type PaginateResponse[T any] struct {
	// Items is the list of items in this page.
	Items []T `json:"items"`

	// Total is the total number of items across all pages.
	Total int64 `json:"total"`

	// Limit is the maximum number of items.
	Limit int `json:"limit"`

	// Offset is the starting position.
	Offset int `json:"offset"`

	// HasMore indicates if there are more items beyond the current page.
	HasMore bool `json:"has_more"`
}

// APIResponse represents a standard API response wrapper.
type APIResponse[T any] struct {
	// Success indicates if the API call was successful.
	Success bool `json:"success"`

	// Data contains the payload of the response.
	Data T `json:"data,omitempty"`

	// Error contains error details if the API call was not successful.
	Error *APIError `json:"error,omitempty"`

	// Meta contains additional metadata about the response.
	Meta *ResponseMeta `json:"meta,omitempty"`
}

// APIError represents error details in an API response.
type APIError struct {
	// Code is the error code.
	Code string `json:"code"`

	// Message is a human-readable error message.
	Message string `json:"message"`

	// Details provides additional information about the error.
	Details map[string]any `json:"details,omitempty"`

	// ValidationErrors contains field-level validation errors.
	ValidationErrors []ValidationError `json:"validation_errors,omitempty"`
}

// ValidationError represents a field validation error.
type ValidationError struct {
	// Field is the field that failed validation.
	Field string `json:"field"`

	// Message is the validation error message.
	Message string `json:"message"`

	// Value is the invalid value (if safe to show).
	Value any `json:"value,omitempty"`
}

// ResponseMeta contains metadata about the response.
type ResponseMeta struct {
	// RequestID is the unique identifier for the request.
	RequestID string `json:"request_id,omitempty"`

	// Timestamp is the time when the response was generated.
	Timestamp string `json:"timestamp,omitempty"`

	// Version is the API version.
	Version string `json:"version,omitempty"`
}

// NewErrorResponse creates a new API error response.
//
// Parameters:
//   - data: The response data
//
// Returns:
//   - APIResponse[T]: The success response wrapper
func NewSuccessResponse[T any](data T) APIResponse[T] {
	return APIResponse[T]{
		Success: true,
		Data:    data,
	}
}

// NewErrorResponse creates a new API error response.
//
// Parameters:
//   - code: The error code
//   - message: The error message
//
// Returns:
//   - APIResponse[T]: The error response wrapper
func NewErrorResponse[T any](code, message string) APIResponse[T] {
	return APIResponse[T]{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
	}
}

// NewValidationErrorResponse creates a new API validation error response.
//
// Parameters:
//   - errors: The list of validation errors
//
// Returns:
//   - APIResponse[T]: The validation error response wrapper
func NewValidationErrorResponse[T any](errors []ValidationError) APIResponse[T] {
	return APIResponse[T]{
		Success: false,
		Error: &APIError{
			Code:             "VALIDATION_ERROR",
			Message:          "Request validation failed",
			ValidationErrors: errors,
		},
	}
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	// Status indicates the health status.
	Status string `json:"status"`

	// Version is the application version.
	Version string `json:"version"`

	// Uptime is how long the service has been running.
	Uptime string `json:"uptime"`

	// Checks contains individual component health check
	Checks map[string]HealthCheckResult `json:"checks"`
}

// HealthCheckResult represents a single health check result.
type HealthCheckResult struct {
	// Status indicates the health status of the component.
	Status string `json:"status"`

	// Message provides additional information about the health status.
	Message string `json:"message,omitempty"`

	// ResponseTime is the time taken to respond in milliseconds.
	ResponseTime int64 `json:"response_time_ms,omitempty"`
}
