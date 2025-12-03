// Package repository contains the repository interfaces and related errors.
package repository

import "errors"

// Repository errors define common error conditions across all repositories.
// These errors are used to communicate specific failure conditions
// from the data access layer to the application layer.

var (
	// ErrUserNotFound is returned when an order cannot be found by ID.
	ErrOrderNotFound = errors.New("order not found")

	// ErrProductNotFound is returned when a product cannot be found by ID or SKU.
	ErrProductNotFound = errors.New("product not found")

	// ErrCustomerNotFound is returned when a customer cannot be found by ID or email.
	ErrCustomerNotFound = errors.New("customer not found")

	// ErrDuplicateEmail is returned when trying to create a customer with
	// an email that already exists.
	ErrDuplicateEmail = errors.New("email already exists")

	// ErrDuplicateSKU is returned when trying to create a product with
	// a SKU that already exists.
	ErrDuplicateSKU = errors.New("SKU already exists")

	// ErrOptimisticLock is returned when an update fails due to
	// a version mismatch (concurrent modification).
	ErrOptimisticLock = errors.New("optimistic lock conflict: record was modified by another transaction")

	// ErrInsufficientStock is returned when trying to deduct more stock
	// than is available.
	ErrInsufficientStock = errors.New("insufficient stock available")

	// ErrConnectionFailed is returned when the database connection fails.
	ErrConnectionFailed = errors.New("database connection failed")

	// ErrTransactionFailed is returned when a database transaction fails.
	ErrTransactionFailed = errors.New("database transaction failed")

	// ErrInvalidInput is returned when repository receives invalid input.
	ErrInvalidInput = errors.New("invalid input provided")
)

// IsNotFoundError checks if the error is a not found error.
// This is useful for handling not-found cases uniformly.
//
// Parameters:
//   - err: error to check
//
// Returns:
//   - bool: true if the error indicates a resource was not found
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrOrderNotFound) ||
		errors.Is(err, ErrProductNotFound) ||
		errors.Is(err, ErrCustomerNotFound)
}

// IsDuplicateError checks if the error is a duplicate entry error.
//
// Parameters:
//   - err: error to check
//
// Returns:
//   - bool: true if the error indicates a duplicate key violation
func IsDuplicateError(err error) bool {
	return errors.Is(err, ErrDuplicateEmail) ||
		errors.Is(err, ErrDuplicateSKU)
}
