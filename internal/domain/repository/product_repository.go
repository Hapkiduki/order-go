// Package repository contains the repository interfaces (ports) for data access.
package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/hapkiduki/order-go/internal/domain/entity"
)

// ProductFilter contains criteria for filtering products.
type ProductFilter struct {
	// Category filters products by category.
	Category *string

	// Status filters products by status.
	Status *entity.ProductStatus

	// MinPrice filters products with price >= this amount (in cents).
	MinPrice *int64

	// MaxPrice filters products with price <= this amount (in cents).
	MaxPrice *int64

	// InStock filters to only products with stock > 0.
	InStock *bool

	// NeedsReorder filters to products below reorder level.
	NeedsReorder *bool

	// Tags filters products that have any of these tags.
	Tags []string

	// SearchTerm searches in name, description, and SKU.
	SearchTerm string

	// Limit specifies the maximum number of results
	Limit int

	// Offset specifies the starting position for pagination
	Offset int

	// SortBy specifies the field to sort by
	SortBy string

	// SortOrder specifies ascending ("asc") or descending ("desc")
	SortOrder string
}

// ProductRepository defines the interface for product persistance operations.
// It abstracts the data access layer for products entities.
//
// Example usage:
//
// repo := postgres.NewProductRepository(db)
// product, err := repo.GetByID(ctx, productID)
type ProductRepository interface {
	// Create persists a new product to the data store.
	//
	// Parameters:
	//   - ctx: context for cancellation and deadlines
	//   - product: The product to create
	//
	// Returns:
	//   - error: any error encountered during creation
	Create(ctx context.Context, product *entity.Product) error

	// GetByID retrieves a product by its unique identifier.
	//
	// Parameters:
	//   - ctx: context for cancellation and deadlines
	//   - id: The product's UUID
	//
	// Returns:
	//   - *entity.Product: The retrieved product, or nil if not found
	//   - error: ErrProductNotFound if product doesn't exist
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Product, error)

	// GetBySKU retrieves a product by its SKU.
	//
	// Parameters:
	//   - ctx: context for cancellation and deadlines
	//   - sku: The product's SKU
	//
	// Returns:
	//   - *entity.Product: The retrieved product, or nil if not found
	//   - error: ErrProductNotFound if product doesn't exist
	GetBySKU(ctx context.Context, sku string) (*entity.Product, error)

	// Update persists changes to an existing product.
	//
	// Parameters:
	//   - ctx: context for cancellation and deadlines
	//   - product: The product to update
	//
	// Returns:
	//   - error: ErrOptimisticLock if version mismatch
	Update(ctx context.Context, product *entity.Product) error

	// Delete removes a product from the data store.
	//
	// Parameters:
	//   - ctx: context for cancellation and deadlines
	//   - id: The product's UUID
	//
	// Returns:
	//   - error: ErrProductNotFound if product doesn't exist
	Delete(ctx context.Context, id uuid.UUID) error

	// FindAll retrieves products matching the given filter criteria.
	//
	// Parameters:
	//   - ctx: context for cancellation and deadlines
	//   - filter: Criteria to filter products
	//
	// Returns:
	//   - []*entity.Product: List of matching products
	//   - error: any error encountered during retrieval
	FindAll(ctx context.Context, filter ProductFilter) ([]*entity.Product, error)

	// FindByCategory retrieves all products in a category.
	//
	// Parameters:
	//   - ctx: context for cancellation and deadlines
	//   - category: The category name
	//
	// Returns:
	//   - []*entity.Product: Slice of products in the category
	//   - error: any error encountered during retrieval
	FindByCategory(ctx context.Context, category string) ([]*entity.Product, error)

	// FindLowStock retrieves products with stock below their reorder level.
	//
	// Parameters:
	//   - ctx: context for cancellation and deadlines
	//
	// Returns:
	//   - []*entity.Product: Slice of low stock products
	//   - error: any error encountered during retrieval
	FindLowStock(ctx context.Context) ([]*entity.Product, error)

	// FindByIDs retrieves products matching the given IDs.
	// Useful for batch operations.
	//
	// Parameters:
	//   - ctx: context for cancellation and deadlines
	//   - ids: Slice of product UUIDs
	//
	// Returns:
	//   - []*entity.Product: Slice of matching products
	//   - error: any error encountered during retrieval
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*entity.Product, error)

	// UpdateStock updates the stock quantity for a product.
	// This operation is an atomic operation to prevent race conditions.
	//
	// Parameters:
	//   - ctx: context for cancellation and deadlines
	//   - id: The product's UUID
	//   - quantity: New stock quantity
	//
	// Returns:
	//   - error: any error that occurred during the update
	UpdateStock(ctx context.Context, id uuid.UUID, quantity int) error

	// DeductStock atomically reduces by the specified amount.
	// Fails if insufficient stock is available.
	//
	// Parameters:
	//   - ctx: context for cancellation and deadlines
	//   - id: The product's UUID
	//   - quantity: Amount to deduct from stock
	//
	// Returns:
	//   - error: ErrInsufficientStock if not enough stock available
	DeductStock(ctx context.Context, id uuid.UUID, quantity int) error

	// Count returns the total number of products matching the filter.
	//
	// Parameters:
	//   - ctx: context for cancellation and deadlines
	//   - filter: Criteria to filter products
	//
	// Returns:
	//   - int64: Count of matching products
	//   - error: any error encountered during counting
	Count(ctx context.Context, filter ProductFilter) (int64, error)

	// ExistsBySKU checks if a product with the given SKU exists.
	//
	// Parameters:
	//   - ctx: context for cancellation and deadlines
	//   - sku: The product's SKU
	//
	// Returns:
	//   - bool: true if product exists, false otherwise
	//   - error: any error encountered during the check
	ExistsBySKU(ctx context.Context, sku string) (bool, error)
}
