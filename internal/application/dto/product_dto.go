package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/hapkiduki/order-go/internal/domain/entity"
)

// ========================================================
// Request DTOs
// ========================================================

// CreateProductRequest contains the data needed to create a new product.
type CreateProductRequest struct {
	// Name is the product display name (required)
	Name string `json:"name" validate:"required,max=200"`

	// Description is the product description (optional)
	Description string `json:"description,omitempty" validate:"max=2000"`

	// SKU is the stock keeping unit(required, unique)
	SKU string `json:"sku" validate:"required,max=50"`

	// Category is the product category (optional)
	Category string `json:"category,omitempty" validate:"max=100"`

	// PriceAmount is the price in smallest currency unit (required)
	PriceAmount int64 `json:"price_amount" validate:"required,min=1"`

	// PriceCurrency is the ISO currency code (defaults to "USD")
	PriceCurrency string `json:"currency,omitempty" validate:"omitempty,len=3"`

	// CostPriceAmount is the cost/purchaseprice (optional)
	CostPriceAmount int64 `json:"cost_price_amount,omitempty" validate:"omitempty,min=0"`

	// StockQuantity is the initial stock level (required)
	StockQuantity int `json:"stock_quantity" validate:"required,min=0"`

	// ReorderLevel is the stock level that triggers reorder (optional)
	ReorderLevel int `json:"reorder_level,omitempty" validate:"min=0"`

	// Weigth in kilograms (optional)
	Weight float64 `json:"weight,omitempty" validate:"min=0"`

	//  ImageUrl is the URL of the product image (optional)
	ImageUrl string `json:"image_url,omitempty" validate:"omitempty,url"`

	//  Tags for categorizations (optional)
	Tags []string `json:"tags,omitempty" validate:"dive,max=50"`
}

type ProductFilterRequest struct{}

// ========================================================
// Response DTOs
// ========================================================
type ProductResponse struct {
	// ID is the product's unique identifier
	ID uuid.UUID `json:"id"`

	// Name is the product's display name
	Name string `json:"name"`

	// Description provides details about the product
	Description string `json:"description"`

	// SKU is the stock keeping unit
	SKU string `json:"sku"`

	// Category classifies the product
	Category string `json:"category"`

	// Price is the selling price
	Price MoneyResponse `json:"price"`

	// CostPrice is the cost/purchase price
	CostPrice *MoneyResponse `json:"cost_price,omitempty"`

	// StockQuantity is the current inventory level
	StockQuantity int `json:"stock_quantity"`

	// ReorderLevel is the stock level that triggers a reorder
	ReorderLevel int `json:"reorder_level"`

	// Status indicates the product availability
	Status string `json:"status"`

	// IsAvailable indicates if the product can be purchased
	IsAvailable bool `json:"is_available"`

	// NeedsReorder indicates if the product is below the reorder level
	NeedsReorder bool `json:"needs_reorder"`

	// Weight is the product's weight in kilograms
	Weight float64 `json:"weight,omitempty"`

	// ImagesURL is the URL of the product image
	ImageURL string `json:"image_url,omitempty"`

	// Tags for categorization
	Tags []string `json:"tags,omitempty"`

	// Margin is the profit margin percentage
	Margin float64 `json:"margin,omitempty"`

	// CreatedAt is the timestamp when the product was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is the timestamp when the product was last updated
	UpdatedAt time.Time `json:"updated_at"`
}

// ========================================================
// Conversion Functions
// ========================================================

// ProductToResponse converts a Product entity to a ProductResponse DTO.
//
// Parameters:
//   - product: The Product entity to convert.
//
// Returns:
//   - *ProductResponse: The converted DTO.
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
