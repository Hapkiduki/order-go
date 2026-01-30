// Package handler contains HTTP request handlers for the API.
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/hapkiduki/order-go/internal/application/dto"
	"github.com/hapkiduki/order-go/internal/application/usecase"
	"github.com/hapkiduki/order-go/internal/domain/repository"
	"github.com/hapkiduki/order-go/pkg/validator"
)

// ProductHandler handles HTTP requests for product operations.
type ProductHandler struct {
	productUseCase *usecase.ProductUseCase
	validator      *validator.Validator
}

// NewProductHandler creates a new ProductHandler with the required dependencies.
//
// Parameters:
//   - productUseCase: The product use case for business logic
//   - validator: Request validator
//
// Returns:
//   - *ProductHandler: The handler instance
func NewProductHandler(productUseCase *usecase.ProductUseCase, validator *validator.Validator) *ProductHandler {
	return &ProductHandler{
		productUseCase: productUseCase,
		validator:      validator,
	}
}

// Routes returns the chi router with all product routes registered.
//
// Returns:
//   - chi.Router: Router with product routes
func (h *ProductHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.CreateProduct)
	r.Get("/", h.ListProducts)
	r.Get("/low-stock", h.GetLowStockProducts)
	r.Get("/{id}", h.GetProduct)
	r.Put("/{id}", h.UpdateProduct)
	r.Patch("/{id}/stock", h.UpdateStock)
	r.Post("/{id}/activate", h.ActivateProduct)
	r.Post("/{id}/deactivate", h.DeactivateProduct)
	r.Delete("/{id}", h.DeleteProduct)

	return r
}

// CreateProduct handles POST /products
// @Summary Create a new product
// @Description Create a new product in the inventory
// @Tags products
// @Accept json
// @Produce json
// @Param product body dto.CreateProductRequest true "Product creation data"
// @Success 201 {object} dto.APIResponse[dto.ProductResponse]
// @Failure 400 {object} dto.APIResponse[any]
// @Failure 409 {object} dto.APIResponse[any]
// @Failure 500 {object} dto.APIResponse[any]
// @Router /products [post]
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateProductRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.NewErrorResponse[any]("INVALID_REQUEST", "Invalid request body"))
		return
	}

	// Set default currency if not provided
	if req.PriceCurrency == "" {
		req.PriceCurrency = "USD"
	}

	// Validate request
	if errors := h.validator.Validate(req); len(errors) > 0 {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.NewValidationErrorResponse[any](errors))
		return
	}

	product, err := h.productUseCase.CreateProduct(r.Context(), req)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, dto.NewSuccessResponse(product))
}

// GetProduct handles GET /products/{id}
// @Summary Get a product by ID
// @Description Retrieve product details by product ID
// @Tags products
// @Produce json
// @Param id path string true "Product ID" format(uuid)
// @Success 200 {object} dto.APIResponse[dto.ProductResponse]
// @Failure 400 {object} dto.APIResponse[any]
// @Failure 404 {object} dto.APIResponse[any]
// @Failure 500 {object} dto.APIResponse[any]
// @Router /products/{id} [get]
func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.NewErrorResponse[any]("INVALID_ID", "Invalid product ID format"))
		return
	}

	product, err := h.productUseCase.GetProduct(r.Context(), id)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, dto.NewSuccessResponse(product))
}

// ListProducts handles GET /products
// @Summary List products
// @Description Retrieve a paginated list of products with optional filters
// @Tags products
// @Produce json
// @Param category query string false "Filter by category"
// @Param status query string false "Filter by status"
// @Param in_stock query bool false "Filter to in-stock products only"
// @Param min_price query int false "Minimum price filter (in cents)"
// @Param max_price query int false "Maximum price filter (in cents)"
// @Param search_term query string false "Search in name, description, and SKU"
// @Param limit query int false "Page size (default 20, max 100)"
// @Param offset query int false "Starting position"
// @Param sort_by query string false "Sort field"
// @Param sort_order query string false "Sort order (asc, desc)"
// @Success 200 {object} dto.APIResponse[dto.PaginatedResponse[dto.ProductResponse]]
// @Failure 400 {object} dto.APIResponse[any]
// @Failure 500 {object} dto.APIResponse[any]
// @Router /products [get]
func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	filter := dto.ProductFilterRequest{
		Limit:     20,
		Offset:    0,
		SortBy:    "name",
		SortOrder: "asc",
	}

	// Parse query parameters
	q := r.URL.Query()

	if category := q.Get("category"); category != "" {
		filter.Category = &category
	}

	if status := q.Get("status"); status != "" {
		filter.Status = &status
	}

	if inStockStr := q.Get("in_stock"); inStockStr != "" {
		inStock := inStockStr == "true"
		filter.InStock = &inStock
	}

	if minPriceStr := q.Get("min_price"); minPriceStr != "" {
		if minPrice, err := strconv.ParseInt(minPriceStr, 10, 64); err == nil {
			filter.MinPrice = &minPrice
		}
	}

	if maxPriceStr := q.Get("max_price"); maxPriceStr != "" {
		if maxPrice, err := strconv.ParseInt(maxPriceStr, 10, 64); err == nil {
			filter.MaxPrice = &maxPrice
		}
	}

	if searchTerm := q.Get("search_term"); searchTerm != "" {
		filter.SearchTerm = searchTerm
	}

	if limitStr := q.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			filter.Limit = limit
		}
	}

	if offsetStr := q.Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	if sortBy := q.Get("sort_by"); sortBy != "" {
		filter.SortBy = sortBy
	}

	if sortOrder := q.Get("sort_order"); sortOrder != "" {
		filter.SortOrder = sortOrder
	}

	// Validate request
	if errors := h.validator.Validate(filter); len(errors) > 0 {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.NewValidationErrorResponse[any](errors))
		return
	}

	result, err := h.productUseCase.ListProducts(r.Context(), filter)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, dto.NewSuccessResponse(result))
}

// UpdateProduct handles PUT /products/{id}
// @Summary Update a product
// @Description Update product details
// @Tags products
// @Accept json
// @Produce json
// @Param id path string true "Product ID" format(uuid)
// @Param product body dto.UpdateProductRequest true "Product update data"
// @Success 200 {object} dto.APIResponse[dto.ProductResponse]
// @Failure 400 {object} dto.APIResponse[any]
// @Failure 404 {object} dto.APIResponse[any]
// @Failure 500 {object} dto.APIResponse[any]
// @Router /products/{id} [put]
func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.NewErrorResponse[any]("INVALID_ID", "Invalid product ID format"))
		return
	}

	var req dto.UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.NewErrorResponse[any]("INVALID_REQUEST", "Invalid request body"))
		return
	}

	// Validate request
	if errors := h.validator.Validate(req); len(errors) > 0 {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.NewValidationErrorResponse[any](errors))
		return
	}

	product, err := h.productUseCase.UpdateProduct(r.Context(), id, req)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, dto.NewSuccessResponse(product))
}

// UpdateStock handles PATCH /products/{id}/stock
// @Summary Update product stock
// @Description Update the stock quantity for a product
// @Tags products
// @Accept json
// @Produce json
// @Param id path string true "Product ID" format(uuid)
// @Param stock body dto.UpdateStockRequest true "Stock update data"
// @Success 200 {object} dto.APIResponse[dto.ProductResponse]
// @Failure 400 {object} dto.APIResponse[any]
// @Failure 404 {object} dto.APIResponse[any]
// @Failure 500 {object} dto.APIResponse[any]
// @Router /products/{id}/stock [patch]
func (h *ProductHandler) UpdateStock(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.NewErrorResponse[any]("INVALID_ID", "Invalid product ID format"))
		return
	}

	var req dto.UpdateStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.NewErrorResponse[any]("INVALID_REQUEST", "Invalid request body"))
		return
	}

	// Validate request
	if errors := h.validator.Validate(req); len(errors) > 0 {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.NewValidationErrorResponse[any](errors))
		return
	}

	product, err := h.productUseCase.UpdateStock(r.Context(), id, req.Quantity)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, dto.NewSuccessResponse(product))
}

// GetLowStockProducts handles GET /products/low-stock
// @Summary Get low stock products
// @Description Retrieve products below their reorder level
// @Tags products
// @Produce json
// @Success 200 {object} dto.APIResponse[[]dto.ProductResponse]
// @Failure 500 {object} dto.APIResponse[any]
// @Router /products/low-stock [get]
func (h *ProductHandler) GetLowStockProducts(w http.ResponseWriter, r *http.Request) {
	products, err := h.productUseCase.GetLowStockProducts(r.Context())
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, dto.NewSuccessResponse(products))
}

// ActivateProduct handles POST /products/{id}/activate
// @Summary Activate a product
// @Description Activate a product, making it available for purchase
// @Tags products
// @Produce json
// @Param id path string true "Product ID" format(uuid)
// @Success 200 {object} dto.APIResponse[dto.ProductResponse]
// @Failure 400 {object} dto.APIResponse[any]
// @Failure 404 {object} dto.APIResponse[any]
// @Failure 500 {object} dto.APIResponse[any]
// @Router /products/{id}/activate [post]
func (h *ProductHandler) ActivateProduct(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.NewErrorResponse[any]("INVALID_ID", "Invalid product ID format"))
		return
	}

	product, err := h.productUseCase.ActivateProduct(r.Context(), id)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, dto.NewSuccessResponse(product))
}

// DeactivateProduct handles POST /products/{id}/deactivate
// @Summary Deactivate a product
// @Description Deactivate a product, making it unavailable for purchase
// @Tags products
// @Produce json
// @Param id path string true "Product ID" format(uuid)
// @Success 200 {object} dto.APIResponse[dto.ProductResponse]
// @Failure 400 {object} dto.APIResponse[any]
// @Failure 404 {object} dto.APIResponse[any]
// @Failure 500 {object} dto.APIResponse[any]
// @Router /products/{id}/deactivate [post]
func (h *ProductHandler) DeactivateProduct(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.NewErrorResponse[any]("INVALID_ID", "Invalid product ID format"))
		return
	}

	product, err := h.productUseCase.DeactivateProduct(r.Context(), id)
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, dto.NewSuccessResponse(product))
}

// DeleteProduct handles DELETE /products/{id}
// @Summary Delete a product
// @Description Remove a product from the inventory
// @Tags products
// @Produce json
// @Param id path string true "Product ID" format(uuid)
// @Success 204 "No Content"
// @Failure 400 {object} dto.APIResponse[any]
// @Failure 404 {object} dto.APIResponse[any]
// @Failure 500 {object} dto.APIResponse[any]
// @Router /products/{id} [delete]
func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.NewErrorResponse[any]("INVALID_ID", "Invalid product ID format"))
		return
	}

	if err := h.productUseCase.DeleteProduct(r.Context(), id); err != nil {
		h.handleError(w, r, err)
		return
	}

	render.Status(r, http.StatusNoContent)
	render.NoContent(w, r)
}

// handleError converts domain errors to HTTP responses.
func (h *ProductHandler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case err == repository.ErrProductNotFound:
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, dto.NewErrorResponse[any]("PRODUCT_NOT_FOUND", "Product not found"))
	case err == repository.ErrDuplicateSKU:
		render.Status(r, http.StatusConflict)
		render.JSON(w, r, dto.NewErrorResponse[any]("DUPLICATE_SKU", "A product with this SKU already exists"))
	case err == repository.ErrOptimisticLock:
		render.Status(r, http.StatusConflict)
		render.JSON(w, r, dto.NewErrorResponse[any]("CONFLICT", "Product was modified by another request"))
	case err == repository.ErrInsufficientStock:
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.NewErrorResponse[any]("INSUFFICIENT_STOCK", "Product has no stock and cannot be activated"))
	default:
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, dto.NewErrorResponse[any]("INTERNAL_ERROR", "An internal error occurred"))
	}
}
