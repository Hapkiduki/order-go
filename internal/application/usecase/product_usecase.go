package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hapkiduki/order-go/internal/application/dto"
	"github.com/hapkiduki/order-go/internal/application/port"
	"github.com/hapkiduki/order-go/internal/domain/entity"
	"github.com/hapkiduki/order-go/internal/domain/repository"
	"github.com/hapkiduki/order-go/internal/domain/valueobject"
)

// ProductUseCase handles business logic related to products.
// It cordinates between the product repository and external services.
type ProductUseCase struct {
	productRepo repository.ProductRepository
	logger      port.Logger
}

// NewProductUseCase creates a new instance of ProductUseCase with the provided dependencies.
//
// Parameters:
//   - repo: Repository for product presistence.
//   - logger: Structured logger.
func NewProductUseCase(
	repo repository.ProductRepository,
	logger port.Logger) *ProductUseCase {
	return &ProductUseCase{
		productRepo: repo,
		logger:      logger,
	}
}

// CreateProduct creates a new product in the inventory.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines.
//   - input: DTO containing product creation details.
//
// Returns:
//   - *dto.ProductResponse: The product details.
//   - error: ErrDuplicateSKU if SKU already exists.
func (uc *ProductUseCase) CreateProduct(ctx context.Context, input dto.CreateProductRequest) (*dto.ProductResponse, error) {
	uc.logger.Info("Creating new product", "name", input.Name, "sku", input.SKU)

	// Check if SKU already exists
	exists, err := uc.productRepo.ExistsBySKU(ctx, input.SKU)
	if err != nil {
		return nil, fmt.Errorf("Failed to chekc SKU: %w", err)
	}
	if exists {
		return nil, repository.ErrDuplicateSKU
	}

	// Create price value object
	price := valueobject.NewMoney(input.PriceAmount, valueobject.Currency(input.PriceCurrency))

	product, err := entity.NewProduct(input.Name, input.SKU, price, input.StockQuantity)
	if err != nil {
		return nil, fmt.Errorf("Failed to create product: %w", err)
	}

	// Set optional fields
	if input.Description != "" {
		product.Description = input.Description
	}
	if input.Category != "" {
		product.Category = input.Category
	}
	if input.CostPriceAmount > 0 {
		product.CostPrice = valueobject.NewMoney(input.CostPriceAmount, valueobject.Currency(input.PriceCurrency))
	}
	if input.ReorderLevel > 0 {
		product.ReorderLevel = input.ReorderLevel
	}
	if input.Weight > 0 {
		product.Weight = input.Weight
	}
	if input.ImageUrl != "" {
		product.ImageURL = input.ImageUrl
	}
	for _, tag := range input.Tags {
		product.AddTag(tag)
	}

	// Persist the product
	if err := uc.productRepo.Create(ctx, product); err != nil {
		uc.logger.Error("Failed to craete product", "error", err)
		return nil, fmt.Errorf("Failed to save product: %w", err)
	}

	// TODO: Invalidate cache

	uc.logger.Info("Product created successfully", "product_id", product.ID)
	return dto.ProductToResponse(product), nil
}

func (uc *ProductUseCase) GetProduct(ctx context.Context, productId uuid.UUID) (*dto.ProductResponse, error) {
	return nil, nil
}

func (uc *ProductUseCase) GetProductBySKU(ctx context.Context, sku string) (*dto.ProductResponse, error) {
	return nil, nil
}

func (uc *ProductUseCase) UpdateStock(ctx context.Context, productId uuid.UUID, quantity int) (*dto.ProductResponse, error) {
	return nil, nil
}

func (uc *ProductUseCase) DeleteProduct(ctx context.Context, productId uuid.UUID) error {
	return nil
}

func (uc *ProductUseCase) ListProducts(ctx context.Context, filter dto.ProductFilterRequest) (*dto.PaginatedResponse[dto.ProductResponse], error) {
	return nil, nil
}

func (uc *ProductUseCase) GetLowStockProducts(ctx context.Context) (*[]dto.ProductResponse, error) {
	return nil, nil
}

func (uc *ProductUseCase) ActivateProduct(ctx context.Context, productId uuid.UUID) (*dto.ProductResponse, error) {
	return nil, nil
}

func (uc *ProductUseCase) DeactivateProduct(ctx context.Context, productId uuid.UUID) (*dto.ProductResponse, error) {
	return nil, nil
}
