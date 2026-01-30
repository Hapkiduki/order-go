// Package dto contains Data Transfer Objects.
package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/hapkiduki/order-go/internal/domain/entity"
)

// =============================================================================
// Request DTOs
// =============================================================================

// CreateProductRequest contains the data needed to create a new product.
type CreateProductRequest struct {
	// Name is the product display name (required)
	Name string `json:"name" validate:"required,max=200"`

	// Description is the product description (optional)
	Description string `json:"description,omitempty" validate:"max=2000"`

	// SKU is the stock keeping unit (required, unique)
	SKU string `json:"sku" validate:"required,max=50"`

	// Category is the product category (optional)
	Category string `json:"category,omitempty" validate:"max=100"`

	// PriceAmount is the price in smallest currency unit (required)
	PriceAmount int64 `json:"price_amount" validate:"required,min=1"`

	// PriceCurrency is the ISO currency code (default: USD)
	PriceCurrency string `json:"price_currency,omitempty" validate:"omitempty,len=3"`

	// CostPriceAmount is the cost/purchase price (optional)
	CostPriceAmount int64 `json:"cost_price_amount,omitempty" validate:"min=0"`

	// StockQuantity is the initial stock level (required)
	StockQuantity int `json:"stock_quantity" validate:"min=0"`

	// ReorderLevel is the stock level that triggers reorder (optional)
	ReorderLevel int `json:"reorder_level,omitempty" validate:"min=0"`

	// Weight in kilograms (optional)
	Weight float64 `json:"weight,omitempty" validate:"min=0"`

	// ImageURL is the product image URL (optional)
	ImageURL string `json:"image_url,omitempty" validate:"omitempty,url"`

	// Tags for categorization (optional)
	Tags []string `json:"tags,omitempty" validate:"dive,max=50"`
}

// UpdateProductRequest contains the data for updating a product.
// All fields are optional - only provided fields will be updated.
type UpdateProductRequest struct {
	// Name is the product display name
	Name *string `json:"name,omitempty" validate:"omitempty,max=200"`

	// Description is the product description
	Description *string `json:"description,omitempty" validate:"omitempty,max=2000"`

	// Category is the product category
	Category *string `json:"category,omitempty" validate:"omitempty,max=100"`

	// PriceAmount is the price in smallest currency unit
	PriceAmount *int64 `json:"price_amount,omitempty" validate:"omitempty,min=1"`

	// PriceCurrency is the ISO currency code
	PriceCurrency *string `json:"price_currency,omitempty" validate:"omitempty,len=3"`

	// ReorderLevel is the stock level that triggers reorder
	ReorderLevel *int `json:"reorder_level,omitempty" validate:"omitempty,min=0"`

	// Weight in kilograms
	Weight *float64 `json:"weight,omitempty" validate:"omitempty,min=0"`

	// ImageURL is the product image URL
	ImageURL *string `json:"image_url,omitempty" validate:"omitempty,url"`
}

// UpdateStockRequest contains data for updating product stock.
type UpdateStockRequest struct {
	// Quantity is the new stock quantity (required)
	Quantity int `json:"quantity" validate:"min=0"`

	// Reason is the reason for the stock change (optional)
	Reason string `json:"reason,omitempty" validate:"max=200"`
}

// ProductFilterRequest contains filter criteria for listing products.
type ProductFilterRequest struct {
	// Category filters by category name
	Category *string `json:"category,omitempty"`

	// Status filters by product status
	Status *string `json:"status,omitempty" validate:"omitempty,oneof=active inactive discontinued out_of_stock"`

	// MinPrice filters products >= this amount
	MinPrice *int64 `json:"min_price,omitempty"`

	// MaxPrice filters products <= this amount
	MaxPrice *int64 `json:"max_price,omitempty"`

	// InStock filters to products with stock > 0
	InStock *bool `json:"in_stock,omitempty"`

	// Tags filters products that have any of these tags
	Tags []string `json:"tags,omitempty"`

	// SearchTerm searches in name, description, and SKU
	SearchTerm string `json:"search_term,omitempty" validate:"max=100"`

	// Limit is the page size (default 20, max 100)
	Limit int `json:"limit,omitempty" validate:"min=0,max=100"`

	// Offset is the starting position
	Offset int `json:"offset,omitempty" validate:"min=0"`

	// SortBy is the field to sort by
	SortBy string `json:"sort_by,omitempty" validate:"omitempty,oneof=name price created_at stock_quantity"`

	// SortOrder is asc or desc
	SortOrder string `json:"sort_order,omitempty" validate:"omitempty,oneof=asc desc"`
}

// =============================================================================
// Response DTOs
// =============================================================================

// ProductResponse represents a product in API responses.
type ProductResponse struct {
	// ID is the product's unique identifier
	ID uuid.UUID `json:"id"`

	// Name is the product display name
	Name string `json:"name"`

	// Description is the product description
	Description string `json:"description,omitempty"`

	// SKU is the stock keeping unit
	SKU string `json:"sku"`

	// Category is the product category
	Category string `json:"category,omitempty"`

	// Price is the selling price
	Price MoneyResponse `json:"price"`

	// CostPrice is the cost/purchase price
	CostPrice *MoneyResponse `json:"cost_price,omitempty"`

	// StockQuantity is the current inventory level
	StockQuantity int `json:"stock_quantity"`

	// ReorderLevel is the stock level that triggers reorder
	ReorderLevel int `json:"reorder_level"`

	// Status indicates the product availability
	Status string `json:"status"`

	// IsAvailable indicates if product can be purchased
	IsAvailable bool `json:"is_available"`

	// NeedsReorder indicates if stock is below reorder level
	NeedsReorder bool `json:"needs_reorder"`

	// Weight in kilograms
	Weight float64 `json:"weight,omitempty"`

	// ImageURL is the product image URL
	ImageURL string `json:"image_url,omitempty"`

	// Tags for categorization
	Tags []string `json:"tags,omitempty"`

	// Margin is the profit margin percentage
	Margin float64 `json:"margin,omitempty"`

	// CreatedAt is when the product was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the product was last updated
	UpdatedAt time.Time `json:"updated_at"`
}

// =============================================================================
// Conversion Functions
// =============================================================================

// ProductToResponse converts a Product entity to a ProductResponse DTO.
//
// Parameters:
//   - product: The product entity to convert
//
// Returns:
//   - *ProductResponse: The converted DTO
func ProductToResponse(product *entity.Product) *ProductResponse {
	response := &ProductResponse{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		SKU:         product.SKU,
		Category:    product.Category,
		Price: MoneyResponse{
			Amount:    product.Price.Amount,
			Currency:  string(product.Price.Currency),
			Formatted: product.Price.Format(),
		},
		StockQuantity: product.StockQuantity,
		ReorderLevel:  product.ReorderLevel,
		Status:        string(product.Status),
		IsAvailable:   product.IsAvailable(),
		NeedsReorder:  product.NeedsReorder(),
		Weight:        product.Weight,
		ImageURL:      product.ImageURL,
		Tags:          product.Tags,
		Margin:        product.CalculateMargin(),
		CreatedAt:     product.CreatedAt,
		UpdatedAt:     product.UpdatedAt,
	}

	if !product.CostPrice.IsZero() {
		response.CostPrice = &MoneyResponse{
			Amount:    product.CostPrice.Amount,
			Currency:  string(product.CostPrice.Currency),
			Formatted: product.CostPrice.Format(),
		}
	}

	return response
}
