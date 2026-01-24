// Package handler contains HTTP request handlers for the API.
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/hapkiduki/order-go/internal/application/dto"
	"github.com/hapkiduki/order-go/internal/application/usecase"
	"github.com/hapkiduki/order-go/pkg/validator"
)

type ProductHandler struct {
	productUseCase *usecase.ProductUseCase
	validator      *validator.Validator
}

func NewProductHandler(
	productUseCase *usecase.ProductUseCase,
	validator *validator.Validator,
) *ProductHandler {
	return &ProductHandler{
		productUseCase: productUseCase,
		validator:      validator,
	}
}

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

func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request)    {}
func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {}

func (h *ProductHandler) UpdateStock(w http.ResponseWriter, r *http.Request) {}

func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {}

func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {}

func (h *ProductHandler) GetLowStockProducts(w http.ResponseWriter, r *http.Request) {}

func (h *ProductHandler) ActivateProduct(w http.ResponseWriter, r *http.Request)   {}
func (h *ProductHandler) DeactivateProduct(w http.ResponseWriter, r *http.Request) {}

func (h *ProductHandler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	panic("unimplemented")
}
