// Package postgres provides PostgreSQL implementations of the repository interfaces.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hapkiduki/order-go/internal/domain/entity"
	"github.com/hapkiduki/order-go/internal/domain/repository"
	"github.com/hapkiduki/order-go/internal/domain/valueobject"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProductRepository is the PostgreSQL implementation of the ProductRepository interface.
type ProductRepository struct {
	pool *pgxpool.Pool
}

// NewProductRepository creates a new ProductRepository with the given connection pool.
//
// Parameters:
//   - pool: PostgreSQL connection pool
//
// Returns:
//   - *ProductRepository: The repository instance
func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{
		pool: pool,
	}
}

// Create persists a new product to the database.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - product: The product to create
//
// Returns:
//   - error: ErrDuplicateSKU if SKU already exists
func (r *ProductRepository) Create(ctx context.Context, product *entity.Product) error {
	query := `
		INSERT INTO products (
			id, name, description, sku, category,
			price_amount, price_currency, cost_price_amount, cost_price_currency,
			stock_quantity, reorder_level, status,
			weight, length, width, height,
			image_url, tags, created_at, updated_at, version
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
		)
	`

	_, err := r.pool.Exec(ctx, query,
		product.ID,
		product.Name,
		product.Description,
		product.SKU,
		product.Category,
		product.Price.Amount,
		product.Price.Currency,
		product.CostPrice.Amount,
		product.CostPrice.Currency,
		product.StockQuantity,
		product.ReorderLevel,
		product.Status,
		product.Weight,
		product.Dimensions.Length,
		product.Dimensions.Width,
		product.Dimensions.Height,
		product.ImageURL,
		product.Tags,
		product.CreatedAt,
		product.UpdatedAt,
		product.Version,
	)

	if err != nil {
		// Check for unique constraint violation on SKU
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return repository.ErrDuplicateSKU
		}
		return fmt.Errorf("failed to insert product: %w", err)
	}

	return nil
}

// GetByID retrieves a product by its unique identifier.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - id: The product's UUID
//
// Returns:
//   - *entity.Product: The found product
//   - error: ErrProductNotFound if product doesn't exist
func (r *ProductRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Product, error) {
	query := `
		SELECT id, name, description, sku, category,
			   price_amount, price_currency, cost_price_amount, cost_price_currency,
			   stock_quantity, reorder_level, status,
			   weight, length, width, height,
			   image_url, tags, created_at, updated_at, version
		FROM products
		WHERE id = $1
	`

	return r.scanProduct(ctx, query, id)
}

// GetBySKU retrieves a product by its SKU.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - sku: The product's SKU
//
// Returns:
//   - *entity.Product: The found product
//   - error: ErrProductNotFound if product doesn't exist
func (r *ProductRepository) GetBySKU(ctx context.Context, sku string) (*entity.Product, error) {
	query := `
		SELECT id, name, description, sku, category,
			   price_amount, price_currency, cost_price_amount, cost_price_currency,
			   stock_quantity, reorder_level, status,
			   weight, length, width, height,
			   image_url, tags, created_at, updated_at, version
		FROM products
		WHERE sku = $1
	`

	return r.scanProduct(ctx, query, sku)
}

// scanProduct is a helper function to scan a product from a query result.
func (r *ProductRepository) scanProduct(ctx context.Context, query string, arg interface{}) (*entity.Product, error) {
	var product entity.Product
	var priceAmount, costPriceAmount int64
	var priceCurrency, costPriceCurrency string
	var length, width, height float64

	err := r.pool.QueryRow(ctx, query, arg).Scan(
		&product.ID,
		&product.Name,
		&product.Description,
		&product.SKU,
		&product.Category,
		&priceAmount,
		&priceCurrency,
		&costPriceAmount,
		&costPriceCurrency,
		&product.StockQuantity,
		&product.ReorderLevel,
		&product.Status,
		&product.Weight,
		&length,
		&width,
		&height,
		&product.ImageURL,
		&product.Tags,
		&product.CreatedAt,
		&product.UpdatedAt,
		&product.Version,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, repository.ErrProductNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query product: %w", err)
	}

	product.Price = valueobject.NewMoney(priceAmount, valueobject.Currency(priceCurrency))
	product.CostPrice = valueobject.NewMoney(costPriceAmount, valueobject.Currency(costPriceCurrency))
	product.Dimensions = valueobject.NewDimensions(length, width, height)

	return &product, nil
}

// Update persists changes to an existing product.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - product: The product to update
//
// Returns:
//   - error: ErrOptimisticLock if version mismatch
func (r *ProductRepository) Update(ctx context.Context, product *entity.Product) error {
	query := `
		UPDATE products SET
			name = $1,
			description = $2,
			category = $3,
			price_amount = $4,
			price_currency = $5,
			cost_price_amount = $6,
			cost_price_currency = $7,
			stock_quantity = $8,
			reorder_level = $9,
			status = $10,
			weight = $11,
			length = $12,
			width = $13,
			height = $14,
			image_url = $15,
			tags = $16,
			updated_at = $17,
			version = version + 1
		WHERE id = $18 AND version = $19
	`

	result, err := r.pool.Exec(ctx, query,
		product.Name,
		product.Description,
		product.Category,
		product.Price.Amount,
		product.Price.Currency,
		product.CostPrice.Amount,
		product.CostPrice.Currency,
		product.StockQuantity,
		product.ReorderLevel,
		product.Status,
		product.Weight,
		product.Dimensions.Length,
		product.Dimensions.Width,
		product.Dimensions.Height,
		product.ImageURL,
		product.Tags,
		time.Now().UTC(),
		product.ID,
		product.Version,
	)
	if err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrOptimisticLock
	}

	product.Version++
	return nil
}

// Delete removes a product from the database.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - id: The product's UUID
//
// Returns:
//   - error: ErrProductNotFound if product doesn't exist
func (r *ProductRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx, "DELETE FROM products WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrProductNotFound
	}

	return nil
}

// FindAll retrieves products matching the given filter criteria.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - filter: Filter criteria for the query
//
// Returns:
//   - []*entity.Product: Slice of matching products
//   - error: Any error that occurred during query
func (r *ProductRepository) FindAll(ctx context.Context, filter repository.ProductFilter) ([]*entity.Product, error) {
	query, args := r.buildFindAllQuery(filter)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer rows.Close()

	var products []*entity.Product
	for rows.Next() {
		var product entity.Product
		var priceAmount, costPriceAmount int64
		var priceCurrency, costPriceCurrency string
		var length, width, height float64

		err := rows.Scan(
			&product.ID,
			&product.Name,
			&product.Description,
			&product.SKU,
			&product.Category,
			&priceAmount,
			&priceCurrency,
			&costPriceAmount,
			&costPriceCurrency,
			&product.StockQuantity,
			&product.ReorderLevel,
			&product.Status,
			&product.Weight,
			&length,
			&width,
			&height,
			&product.ImageURL,
			&product.Tags,
			&product.CreatedAt,
			&product.UpdatedAt,
			&product.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}

		product.Price = valueobject.NewMoney(priceAmount, valueobject.Currency(priceCurrency))
		product.CostPrice = valueobject.NewMoney(costPriceAmount, valueobject.Currency(costPriceCurrency))
		product.Dimensions = valueobject.NewDimensions(length, width, height)

		products = append(products, &product)
	}

	return products, nil
}

// buildFindAllQuery constructs the SQL query and arguments for FindAll.
func (r *ProductRepository) buildFindAllQuery(filter repository.ProductFilter) (string, []interface{}) {
	query := `
		SELECT id, name, description, sku, category,
			   price_amount, price_currency, cost_price_amount, cost_price_currency,
			   stock_quantity, reorder_level, status,
			   weight, length, width, height,
			   image_url, tags, created_at, updated_at, version
		FROM products
		WHERE 1=1
	`
	args := make([]interface{}, 0)
	argIndex := 1

	if filter.Category != nil {
		query += fmt.Sprintf(" AND category = $%d", argIndex)
		args = append(args, *filter.Category)
		argIndex++
	}

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filter.Status)
		argIndex++
	}

	if filter.MinPrice != nil {
		query += fmt.Sprintf(" AND price_amount >= $%d", argIndex)
		args = append(args, *filter.MinPrice)
		argIndex++
	}

	if filter.MaxPrice != nil {
		query += fmt.Sprintf(" AND price_amount <= $%d", argIndex)
		args = append(args, *filter.MaxPrice)
		argIndex++
	}

	if filter.InStock != nil && *filter.InStock {
		query += " AND stock_quantity > 0"
	}

	if filter.NeedsReorder != nil && *filter.NeedsReorder {
		query += " AND stock_quantity <= reorder_level"
	}

	if filter.SearchTerm != "" {
		query += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d OR sku ILIKE $%d)", argIndex, argIndex, argIndex)
		args = append(args, "%"+filter.SearchTerm+"%")
		argIndex++
	}

	if len(filter.Tags) > 0 {
		query += fmt.Sprintf(" AND tags && $%d", argIndex)
		args = append(args, filter.Tags)
		argIndex++
	}

	// Sorting
	sortBy := "created_at"
	if filter.SortBy != "" {
		sortBy = filter.SortBy
	}
	sortOrder := "DESC"
	if filter.SortOrder == "asc" {
		sortOrder = "ASC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

	// Pagination
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
	}

	return query, args
}

// FindByCategory retrieves all products in a category.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - category: The category name
//
// Returns:
//   - []*entity.Product: Slice of products in category
//   - error: Any error that occurred during query
func (r *ProductRepository) FindByCategory(ctx context.Context, category string) ([]*entity.Product, error) {
	filter := repository.ProductFilter{
		Category: &category,
	}
	return r.FindAll(ctx, filter)
}

// FindLowStock retrieves products below their reorder level.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//
// Returns:
//   - []*entity.Product: Slice of products needing reorder
//   - error: Any error that occurred during query
func (r *ProductRepository) FindLowStock(ctx context.Context) ([]*entity.Product, error) {
	needsReorder := true
	filter := repository.ProductFilter{
		NeedsReorder: &needsReorder,
	}
	return r.FindAll(ctx, filter)
}

// FindByIDs retrieves multiple products by their IDs.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - ids: Slice of product UUIDs
//
// Returns:
//   - []*entity.Product: Slice of found products
//   - error: Any error that occurred during query
func (r *ProductRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*entity.Product, error) {
	if len(ids) == 0 {
		return []*entity.Product{}, nil
	}

	query := `
		SELECT id, name, description, sku, category,
			   price_amount, price_currency, cost_price_amount, cost_price_currency,
			   stock_quantity, reorder_level, status,
			   weight, length, width, height,
			   image_url, tags, created_at, updated_at, version
		FROM products
		WHERE id = ANY($1)
	`

	rows, err := r.pool.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer rows.Close()

	var products []*entity.Product
	for rows.Next() {
		var product entity.Product
		var priceAmount, costPriceAmount int64
		var priceCurrency, costPriceCurrency string
		var length, width, height float64

		err := rows.Scan(
			&product.ID,
			&product.Name,
			&product.Description,
			&product.SKU,
			&product.Category,
			&priceAmount,
			&priceCurrency,
			&costPriceAmount,
			&costPriceCurrency,
			&product.StockQuantity,
			&product.ReorderLevel,
			&product.Status,
			&product.Weight,
			&length,
			&width,
			&height,
			&product.ImageURL,
			&product.Tags,
			&product.CreatedAt,
			&product.UpdatedAt,
			&product.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}

		product.Price = valueobject.NewMoney(priceAmount, valueobject.Currency(priceCurrency))
		product.CostPrice = valueobject.NewMoney(costPriceAmount, valueobject.Currency(costPriceCurrency))
		product.Dimensions = valueobject.NewDimensions(length, width, height)

		products = append(products, &product)
	}

	return products, nil
}

// UpdateStock updates the stock quantity for a product.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - id: The product's UUID
//   - quantity: The new stock quantity
//
// Returns:
//   - error: Any error that occurred during update
func (r *ProductRepository) UpdateStock(ctx context.Context, id uuid.UUID, quantity int) error {
	query := `
		UPDATE products 
		SET stock_quantity = stock_quantity + $1, 
			updated_at = $2,
			status = CASE 
				WHEN stock_quantity + $1 > 0 THEN 'active'::text 
				ELSE 'out_of_stock'::text 
			END
		WHERE id = $3
	`

	result, err := r.pool.Exec(ctx, query, quantity, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("failed to update stock: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrProductNotFound
	}

	return nil
}

// DeductStock atomically reduces stock by the specified amount.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - id: The product's UUID
//   - quantity: Amount to deduct
//
// Returns:
//   - error: ErrInsufficientStock if not enough stock
func (r *ProductRepository) DeductStock(ctx context.Context, id uuid.UUID, quantity int) error {
	query := `
		UPDATE products 
		SET stock_quantity = stock_quantity - $1,
			updated_at = $2,
			status = CASE 
				WHEN stock_quantity - $1 > 0 THEN 'active'::text 
				ELSE 'out_of_stock'::text 
			END
		WHERE id = $3 AND stock_quantity >= $1
	`

	result, err := r.pool.Exec(ctx, query, quantity, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("failed to deduct stock: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrInsufficientStock
	}

	return nil
}

// Count returns the total number of products matching the filter.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - filter: Filter criteria for the count
//
// Returns:
//   - int64: Total count of matching products
//   - error: Any error that occurred during query
func (r *ProductRepository) Count(ctx context.Context, filter repository.ProductFilter) (int64, error) {
	query := "SELECT COUNT(*) FROM products WHERE 1=1"
	args := make([]interface{}, 0)
	argIndex := 1

	if filter.Category != nil {
		query += fmt.Sprintf(" AND category = $%d", argIndex)
		args = append(args, *filter.Category)
		argIndex++
	}

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filter.Status)
		argIndex++
	}

	if filter.InStock != nil && *filter.InStock {
		query += " AND stock_quantity > 0"
	}

	var count int64
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count products: %w", err)
	}

	return count, nil
}

// ExistsBySKU checks if a product with the given SKU exists.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - sku: The SKU to check
//
// Returns:
//   - bool: true if product exists
//   - error: Any error that occurred during query
func (r *ProductRepository) ExistsBySKU(ctx context.Context, sku string) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM products WHERE sku = $1)"

	var exists bool
	err := r.pool.QueryRow(ctx, query, sku).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check product existence: %w", err)
	}

	return exists, nil
}
