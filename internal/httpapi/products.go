package httpapi

import (
	"net/http"

	"marketplace/internal/api/generated"
	"marketplace/internal/apperror"
	"marketplace/internal/domain"
	"marketplace/internal/money"
)

func (s *Server) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var body generated.ProductCreate
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, err)
		return
	}

	draft, err := productDraft(body.Name, body.Description, body.Price, body.Stock, body.Category, body.Status)
	if err != nil {
		writeError(w, r, err)
		return
	}
	product, err := s.products.Create(r.Context(), mustActor(r.Context()), draft)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, productResponse(product))
}

func (s *Server) GetProduct(w http.ResponseWriter, r *http.Request, id int64) {
	product, err := s.products.Get(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, productResponse(product))
}

func (s *Server) ListProducts(w http.ResponseWriter, r *http.Request, params generated.ListProductsParams) {
	filter := domain.ProductFilter{Page: 0, Size: 20}
	if params.Page != nil {
		filter.Page = *params.Page
	}
	if params.Size != nil {
		filter.Size = *params.Size
	}
	if params.Status != nil {
		status := domain.ProductStatus(*params.Status)
		filter.Status = &status
	}
	filter.Category = params.Category

	page, err := s.products.List(r.Context(), filter)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, productPageResponse(page))
}

func (s *Server) UpdateProduct(w http.ResponseWriter, r *http.Request, id int64) {
	var body generated.ProductUpdate
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, err)
		return
	}

	draft, err := productDraft(body.Name, body.Description, body.Price, body.Stock, body.Category, body.Status)
	if err != nil {
		writeError(w, r, err)
		return
	}
	product, err := s.products.Update(r.Context(), mustActor(r.Context()), id, draft)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, productResponse(product))
}

func (s *Server) ArchiveProduct(w http.ResponseWriter, r *http.Request, id int64) {
	if err := s.products.Archive(r.Context(), mustActor(r.Context()), id); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func productDraft(
	name string,
	description *string,
	price float64,
	stock int,
	category string,
	status generated.ProductStatus,
) (domain.ProductDraft, error) {
	priceCents, err := money.FromFloat(price)
	if err != nil {
		return domain.ProductDraft{}, apperror.Validation(map[string]any{"price": "must have at most two decimal places"})
	}
	return domain.ProductDraft{
		Name:        name,
		Description: description,
		Price:       priceCents,
		Stock:       stock,
		Category:    category,
		Status:      domain.ProductStatus(status),
	}, nil
}
