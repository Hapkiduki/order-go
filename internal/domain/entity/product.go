// Package entity contains the core bussiness entities of the domain layer.
package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/hapkiduki/order-go/internal/domain/valueobject"
)

// Product errors define domain-specific error conditions for products.
var (
	ErrInvalidProductName    = errors.New("product name cannot be empty")
	ErrInvalidProductSKU     = errors.New("product SKU cannot be empty")
	ErrInvalidProductPrice   = errors.New("product price must be positive")
	ErrInsufficientStock     = errors.New("insufficient stock available")
	ErrNegativeStockQuantity = errors.New("stock quantity cannot be negative")
	ErrProductNotActive      = errors.New("product is not active")
)

// ProductStatus represents the availability status of a product.
type ProductStatus string

const (
	ProductStatusActive       ProductStatus = "active"       // Product is available for purchase
	ProductStatusInactive     ProductStatus = "inactive"     // Product is temporarily unavailable
	ProductStatusDiscontinued ProductStatus = "discontinued" // Product is permanently unavailable
	ProductStatusOutOfStock   ProductStatus = "out_of_stock" // Product has no inventory
)

type Product struct {
	// ID is the unique identifier for the product
	ID uuid.UUID `json:"id"`

	// Name is the name of the product
	Name string `json:"name"`

	// Description provides details about the product
	Description string `json:"description"`

	// SKU is the stock keeping unit identifier
	SKU string `json:"sku"`

	// Category classifies the product
	Category string `json:"category"`

	// Price is the selling price of the product
	Price valueobject.Money `json:"price"`

	// CostPrice is the cost/purchase price (for margin calculation)
	CostPrice valueobject.Money `json:"cost_price"`

	// StockQuantity is the current inventory level
	StockQuantity int `json:"stock_quantity"`

	// ReorderLevel is the stock level that triggers a reorder
	ReorderLevel int `json:"reorder_level"`

	// Status indicates the product availability
	Status ProductStatus `json:"status"`

	// Weight in kilograms (for shipping calculations)
	Weight float64 `json:"weight"`

	// Dimensions for shipping calculations
	Dimensions valueobject.Dimensions `json:"dimensions"`

	// ImageURL is the primary product image
	ImageURL string `json:"image_url,omitempty"`

	// Tags for categorization and search
	Tags []string `json:"tags,omitempty"`

	// CreatedAt is the timestamp when the product was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is the timestamp when the product was last updated
	UpdatedAt time.Time `json:"updated_at"`

	// Version is used for optimistic locking
	Version int `json:"version"`
}

// NewProduct creates a new Product entity with the provided details.
// It validates the input and initializes the product in active state.
//
// Parameters:
//   - name: Name of the product (required)
//   - sku: Stock Keeping Unit identifier (required, must be unique)
//   - price: Selling price of the product (must be positive)
//   - stockQuantity: Initial stock quantity (must be non-negative)
//
// Returns:
//   - *Product: newly created Product
//   - error: VAlidation error if input is invalid
func NewProduct(
	name, sku string,
	price valueobject.Money,
	stockQuantity int,
) (*Product, error) {
	if name == "" {
		return nil, ErrInvalidProductName
	}
	if sku == "" {
		return nil, ErrInvalidProductSKU
	}
	if !price.IsPositive() {
		return nil, ErrInvalidProductPrice
	}
	if stockQuantity < 0 {
		return nil, ErrNegativeStockQuantity
	}

	now := time.Now().UTC()

	status := ProductStatusActive
	if stockQuantity == 0 {
		status = ProductStatusOutOfStock
	}

	return &Product{
		ID:            uuid.New(),
		Name:          name,
		SKU:           sku,
		Price:         price,
		StockQuantity: stockQuantity,
		Status:        status,
		ReorderLevel:  10, // default reorder level
		CreatedAt:     now,
		UpdatedAt:     now,
		Version:       1,
		Tags:          make([]string, 0),
	}, nil
}

// DeductStock decreases the product's stock quantity.
// This is used when items are sold or reserved.
//
// Parameters:
//   - quantity: amount to deduct from stock (must be non-negative)
//
// Returns:
//   - error: ErrNegativeStockQuantity if quantity is negative
//     ErrInsufficientStock if not enough stock available
func (p *Product) DeductStock(quantity int) error {
	if quantity < 0 {
		return ErrNegativeStockQuantity
	}
	if p.StockQuantity < quantity {
		return ErrInsufficientStock
	}
	p.StockQuantity -= quantity
	p.UpdatedAt = time.Now().UTC()

	// Update status if out of stock
	if p.StockQuantity == 0 {
		p.Status = ProductStatusOutOfStock
	}
	return nil
}

// AddStock increases the product's stock quantity.
// This is used when inventory is replenished.
//
// Parameters:
//   - quantity: amount to add to stock (must be non-negative)
//
// Returns:
//   - error: ErrNegativeStockQuantity if quantity is negative
func (p *Product) AddStock(quantity int) error {
	if quantity < 0 {
		return ErrNegativeStockQuantity
	}
	p.StockQuantity += quantity
	p.UpdatedAt = time.Now().UTC()

	// Update status if previously out of stock
	if p.Status == ProductStatusOutOfStock && p.StockQuantity > 0 {
		p.Status = ProductStatusActive
	}
	return nil
}

// SetPrice updates the product's selling price.
//
// Parameters:
//   - price: new selling price (must be positive)
//
// Returns:
//   - error: ErrInvalidProductPrice if price is not positive
func (p *Product) SetPrice(price valueobject.Money) error {
	if !price.IsPositive() {
		return ErrInvalidProductPrice
	}
	p.Price = price
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// Activate sets the product status to active.
// The product must have stock to be activated.
//
// Returns:
//   - error: ErrInsufficientStock if the product has no stock
func (p *Product) Activate() error {
	if p.StockQuantity == 0 {
		return ErrInsufficientStock
	}

	p.Status = ProductStatusActive
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// Deactivate sets the product status to inactive.
// Inactive products are not available for purchase.
func (p *Product) Deactivate() {
	p.Status = ProductStatusInactive
	p.UpdatedAt = time.Now().UTC()
}

// Discontinue permanently marks the product as unavailable.
// Discontinued products cannot be reactivated.
func (p *Product) Discontinue() {
	p.Status = ProductStatusDiscontinued
	p.UpdatedAt = time.Now().UTC()
}

// IsAvailable checks if the product can be purchased.
//
// Returns:
//   - bool: true if product is active and has stock
func (p *Product) IsAvailable() bool {
	return p.Status == ProductStatusActive && p.StockQuantity > 0
}

// IsAvailableForQuantity checks if the specified quantity can be ordered.
//
// Parameters:
//   - quantity: desired quantity to order
//
// Returns:
//   - bool: true if product is available and has sufficient stock
func (p *Product) IsAvailableForQuantity(quantity int) bool {
	return p.IsAvailable() && p.StockQuantity >= quantity
}

// NeedsReorder checks if the product stock is at or below the reorder level.
//
// Returns:
//   - bool: true if stock quantity is less than or equal to reorder level
func (p *Product) NeedsReorder() bool {
	return p.StockQuantity <= p.ReorderLevel
}

// CalculateMargin computes the profit margin as a percentage.
//
// Returns:
//   - float64: Margin percentage (e.g., 25.5 for 25.5%)
func (p *Product) CalculateMargin() float64 {
	if p.CostPrice.IsZero() {
		return 0
	}
	profit := p.Price.Subtract(p.CostPrice)
	return (float64(profit.Amount) / float64(p.Price.Amount)) * 100
}

// UpdateDetails updates the product's descriptive information.
//
// Parameters:
//   - name: new product name (optional, empty string to keep current)
//   - description: new description (optional)
//   - category: new category (optional)
func (p *Product) UpdateDetails(name, description, category string) error {
	if name != "" {
		p.Name = name
	}
	if description != "" {
		p.Description = description
	}
	if category != "" {
		p.Category = category
	}
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// SetDimensions updates the product's physical dimensions.
//
// Parameters:
//   - dimensions: new Dimensions value object
func (p *Product) SetDimensions(dimensions valueobject.Dimensions) {
	p.Dimensions = dimensions
	p.UpdatedAt = time.Now().UTC()
}

// SetWeight updates the product's weight.
//
// Parameters:
//   - weight: new weight in kilograms
func (p *Product) SetWeight(weight float64) {
	p.Weight = weight
	p.UpdatedAt = time.Now().UTC()
}

// AddTag adds a new tag to the product for categorization.
//
// Parameters:
//   - tag: tag to be added
func (p *Product) AddTag(tag string) {
	// Avoid duplicates
	for _, t := range p.Tags {
		if t == tag {
			return
		}
	}
	p.Tags = append(p.Tags, tag)
	p.UpdatedAt = time.Now().UTC()
}

// RemoveTag removes a tag from the product's tag list.
//
// Parameters:
//   - tag: tag to be removed
func (p *Product) RemoveTag(tag string) {
	for i, t := range p.Tags {
		if t == tag {
			p.Tags = append(p.Tags[:i], p.Tags[i+1:]...)
			p.UpdatedAt = time.Now().UTC()
			break
		}
	}
}

// SetReorderLevel updates the reorder threshold.
//
// Parameters:
//   - level: new reorder level (must be non-negative)
func (p *Product) SetReorderLevel(level int) error {
	if level < 0 {
		return ErrNegativeStockQuantity
	}
	p.ReorderLevel = level
	p.UpdatedAt = time.Now().UTC()
	return nil
}
