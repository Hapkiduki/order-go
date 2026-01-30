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

// ProductUseCase handles all product-related business operations.
// It coordinates between the product repository and external services.
type ProductUseCase struct {
	productRepo repository.ProductRepository
	cache       port.CacheService
	logger      port.Logger
}

// NewProductUseCase creates a new ProductUseCase with the required dependencies.
//
// Parameters:
//   - productRepo: Repository for product persistence
//   - cache: Cache service for performance optimization
//   - logger: Structured logger
//
// Returns:
//   - *ProductUseCase: The configured use case instance
func NewProductUseCase(
	productRepo repository.ProductRepository,
	cache port.CacheService,
	logger port.Logger,
) *ProductUseCase {
	return &ProductUseCase{
		productRepo: productRepo,
		cache:       cache,
		logger:      logger,
	}
}

// CreateProduct creates a new product in the inventory.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - input: DTO containing product creation details
//
// Returns:
//   - *dto.ProductResponse: The created product details
//   - error: ErrDuplicateSKU if SKU already exists
func (uc *ProductUseCase) CreateProduct(ctx context.Context, input dto.CreateProductRequest) (*dto.ProductResponse, error) {
	uc.logger.Info("Creating new product", "name", input.Name, "sku", input.SKU)

	// Check if SKU already exists
	exists, err := uc.productRepo.ExistsBySKU(ctx, input.SKU)
	if err != nil {
		return nil, fmt.Errorf("failed to check SKU: %w", err)
	}
	if exists {
		return nil, repository.ErrDuplicateSKU
	}

	// Create price value object
	price := valueobject.NewMoney(input.PriceAmount, valueobject.Currency(input.PriceCurrency))

	// Create the product entity
	product, err := entity.NewProduct(input.Name, input.SKU, price, input.StockQuantity)
	if err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
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
	if input.ImageURL != "" {
		product.ImageURL = input.ImageURL
	}
	for _, tag := range input.Tags {
		product.AddTag(tag)
	}

	// Persist the product
	if err := uc.productRepo.Create(ctx, product); err != nil {
		uc.logger.Error("Failed to create product", "error", err)
		return nil, fmt.Errorf("failed to save product: %w", err)
	}

	// Invalidate product list cache
	if uc.cache != nil {
		_ = uc.cache.Delete(ctx, "products:list")
	}

	uc.logger.Info("Product created successfully", "product_id", product.ID)

	return dto.ProductToResponse(product), nil
}

// GetProduct retrieves a product by its ID.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - productID: The product's UUID
//
// Returns:
//   - *dto.ProductResponse: The product details
//   - error: ErrProductNotFound if product doesn't exist
func (uc *ProductUseCase) GetProduct(ctx context.Context, productID uuid.UUID) (*dto.ProductResponse, error) {
	uc.logger.Debug("Getting product", "product_id", productID)

	// Try cache first
	if uc.cache != nil {
		cacheKey := fmt.Sprintf("product:%s", productID)
		var cached dto.ProductResponse
		if err := uc.cache.Get(ctx, cacheKey, &cached); err == nil {
			return &cached, nil
		}
	}

	// Fetch from database
	product, err := uc.productRepo.GetByID(ctx, productID)
	if err != nil {
		return nil, err
	}

	response := dto.ProductToResponse(product)

	// Cache the result
	if uc.cache != nil {
		cacheKey := fmt.Sprintf("product:%s", productID)
		_ = uc.cache.Set(ctx, cacheKey, response, 600) // 10 minutes TTL
	}

	return response, nil
}

// GetProductBySKU retrieves a product by its SKU.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - sku: The product's SKU
//
// Returns:
//   - *dto.ProductResponse: The product details
//   - error: ErrProductNotFound if product doesn't exist
func (uc *ProductUseCase) GetProductBySKU(ctx context.Context, sku string) (*dto.ProductResponse, error) {
	uc.logger.Debug("Getting product by SKU", "sku", sku)

	product, err := uc.productRepo.GetBySKU(ctx, sku)
	if err != nil {
		return nil, err
	}

	return dto.ProductToResponse(product), nil
}

// UpdateProduct updates an existing product.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - productID: The product's UUID
//   - input: DTO containing update details
//
// Returns:
//   - *dto.ProductResponse: The updated product details
//   - error: ErrProductNotFound if product doesn't exist
func (uc *ProductUseCase) UpdateProduct(ctx context.Context, productID uuid.UUID, input dto.UpdateProductRequest) (*dto.ProductResponse, error) {
	uc.logger.Info("Updating product", "product_id", productID)

	product, err := uc.productRepo.GetByID(ctx, productID)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if input.Name != nil {
		product.Name = *input.Name
	}
	if input.Description != nil {
		product.Description = *input.Description
	}
	if input.Category != nil {
		product.Category = *input.Category
	}
	if input.PriceAmount != nil {
		currency := product.Price.Currency
		if input.PriceCurrency != nil {
			currency = valueobject.Currency(*input.PriceCurrency)
		}
		if err := product.SetPrice(valueobject.NewMoney(*input.PriceAmount, currency)); err != nil {
			return nil, fmt.Errorf("invalid price: %w", err)
		}
	}
	if input.ReorderLevel != nil {
		if err := product.SetReorderLevel(*input.ReorderLevel); err != nil {
			return nil, fmt.Errorf("invalid reorder level: %w", err)
		}
	}
	if input.Weight != nil {
		product.SetWeight(*input.Weight)
	}
	if input.ImageURL != nil {
		product.ImageURL = *input.ImageURL
	}

	// Persist changes
	if err := uc.productRepo.Update(ctx, product); err != nil {
		return nil, fmt.Errorf("failed to update product: %w", err)
	}

	// Invalidate cache
	if uc.cache != nil {
		_ = uc.cache.Delete(ctx, fmt.Sprintf("product:%s", productID))
		_ = uc.cache.Delete(ctx, "products:list")
	}

	uc.logger.Info("Product updated successfully", "product_id", productID)

	return dto.ProductToResponse(product), nil
}

// UpdateStock updates the stock quantity for a product.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - productID: The product's UUID
//   - quantity: The new stock quantity
//
// Returns:
//   - *dto.ProductResponse: The updated product details
//   - error: ErrProductNotFound if product doesn't exist
func (uc *ProductUseCase) UpdateStock(ctx context.Context, productID uuid.UUID, quantity int) (*dto.ProductResponse, error) {
	uc.logger.Info("Updating product stock", "product_id", productID, "quantity", quantity)

	product, err := uc.productRepo.GetByID(ctx, productID)
	if err != nil {
		return nil, err
	}

	previousQty := product.StockQuantity

	if quantity > product.StockQuantity {
		// Adding stock
		if err := product.AddStock(quantity - product.StockQuantity); err != nil {
			return nil, fmt.Errorf("failed to add stock: %w", err)
		}
	} else if quantity < product.StockQuantity {
		// Deducting stock
		if err := product.DeductStock(product.StockQuantity - quantity); err != nil {
			return nil, fmt.Errorf("failed to deduct stock: %w", err)
		}
	}

	if err := uc.productRepo.Update(ctx, product); err != nil {
		return nil, fmt.Errorf("failed to update product: %w", err)
	}

	// Check if we need to raise low stock event
	if product.NeedsReorder() && previousQty > product.ReorderLevel {
		// Stock just dropped below reorder level
		uc.logger.Warn("Product stock is low", "product_id", productID, "stock", product.StockQuantity)
	}

	// Invalidate cache
	if uc.cache != nil {
		_ = uc.cache.Delete(ctx, fmt.Sprintf("product:%s", productID))
	}

	return dto.ProductToResponse(product), nil
}

// DeleteProduct removes a product from the inventory.
// Consider using Discontinue() for soft delete.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - productID: The product's UUID
//
// Returns:
//   - error: ErrProductNotFound if product doesn't exist
func (uc *ProductUseCase) DeleteProduct(ctx context.Context, productID uuid.UUID) error {
	uc.logger.Info("Deleting product", "product_id", productID)

	if err := uc.productRepo.Delete(ctx, productID); err != nil {
		return err
	}

	// Invalidate cache
	if uc.cache != nil {
		_ = uc.cache.Delete(ctx, fmt.Sprintf("product:%s", productID))
		_ = uc.cache.Delete(ctx, "products:list")
	}

	return nil
}

// ListProducts retrieves products based on filter criteria.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - filter: Filter criteria for the query
//
// Returns:
//   - *dto.PaginatedResponse[dto.ProductResponse]: Paginated list of products
//   - error: Any error that occurred during retrieval
func (uc *ProductUseCase) ListProducts(ctx context.Context, filter dto.ProductFilterRequest) (*dto.PaginatedResponse[dto.ProductResponse], error) {
	uc.logger.Debug("Listing products", "filter", filter)

	// Convert DTO filter to repository filter
	repoFilter := repository.ProductFilter{
		Category:   filter.Category,
		SearchTerm: filter.SearchTerm,
		Tags:       filter.Tags,
		Limit:      filter.Limit,
		Offset:     filter.Offset,
		SortBy:     filter.SortBy,
		SortOrder:  filter.SortOrder,
	}

	if filter.Status != nil {
		status := entity.ProductStatus(*filter.Status)
		repoFilter.Status = &status
	}
	if filter.InStock != nil {
		repoFilter.InStock = filter.InStock
	}
	if filter.MinPrice != nil {
		repoFilter.MinPrice = filter.MinPrice
	}
	if filter.MaxPrice != nil {
		repoFilter.MaxPrice = filter.MaxPrice
	}

	products, err := uc.productRepo.FindAll(ctx, repoFilter)
	if err != nil {
		return nil, err
	}

	total, err := uc.productRepo.Count(ctx, repoFilter)
	if err != nil {
		return nil, err
	}

	items := make([]dto.ProductResponse, 0, len(products))
	for _, product := range products {
		items = append(items, *dto.ProductToResponse(product))
	}

	return &dto.PaginatedResponse[dto.ProductResponse]{
		Items:   items,
		Total:   total,
		Limit:   filter.Limit,
		Offset:  filter.Offset,
		HasMore: int64(filter.Offset+len(items)) < total,
	}, nil
}

// GetLowStockProducts retrieves products below their reorder level.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//
// Returns:
//   - []*dto.ProductResponse: List of products needing reorder
//   - error: Any error that occurred during retrieval
func (uc *ProductUseCase) GetLowStockProducts(ctx context.Context) ([]*dto.ProductResponse, error) {
	uc.logger.Debug("Getting low stock products")

	products, err := uc.productRepo.FindLowStock(ctx)
	if err != nil {
		return nil, err
	}

	responses := make([]*dto.ProductResponse, 0, len(products))
	for _, product := range products {
		responses = append(responses, dto.ProductToResponse(product))
	}

	return responses, nil
}

// ActivateProduct activates a product, making it available for purchase.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - productID: The product's UUID
//
// Returns:
//   - *dto.ProductResponse: The updated product details
//   - error: ErrInsufficientStock if product has no stock
func (uc *ProductUseCase) ActivateProduct(ctx context.Context, productID uuid.UUID) (*dto.ProductResponse, error) {
	uc.logger.Info("Activating product", "product_id", productID)

	product, err := uc.productRepo.GetByID(ctx, productID)
	if err != nil {
		return nil, err
	}

	if err := product.Activate(); err != nil {
		return nil, fmt.Errorf("cannot activate product: %w", err)
	}

	if err := uc.productRepo.Update(ctx, product); err != nil {
		return nil, fmt.Errorf("failed to update product: %w", err)
	}

	// Invalidate cache
	if uc.cache != nil {
		_ = uc.cache.Delete(ctx, fmt.Sprintf("product:%s", productID))
	}

	return dto.ProductToResponse(product), nil
}

// DeactivateProduct deactivates a product, making it unavailable.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - productID: The product's UUID
//
// Returns:
//   - *dto.ProductResponse: The updated product details
//   - error: ErrProductNotFound if product doesn't exist
func (uc *ProductUseCase) DeactivateProduct(ctx context.Context, productID uuid.UUID) (*dto.ProductResponse, error) {
	uc.logger.Info("Deactivating product", "product_id", productID)

	product, err := uc.productRepo.GetByID(ctx, productID)
	if err != nil {
		return nil, err
	}

	product.Deactivate()

	if err := uc.productRepo.Update(ctx, product); err != nil {
		return nil, fmt.Errorf("failed to update product: %w", err)
	}

	// Invalidate cache
	if uc.cache != nil {
		_ = uc.cache.Delete(ctx, fmt.Sprintf("product:%s", productID))
	}

	return dto.ProductToResponse(product), nil
}
