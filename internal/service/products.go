package service

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"marketplace/internal/apperror"
	"marketplace/internal/domain"
	"marketplace/internal/requestctx"
	"marketplace/internal/store"
)

type ProductService struct {
	queries *store.Queries
}

func NewProductService(queries *store.Queries) *ProductService {
	return &ProductService{queries: queries}
}

func (s *ProductService) Create(ctx context.Context, actor requestctx.Actor, draft domain.ProductDraft) (domain.Product, error) {
	if actor.Role != domain.RoleSeller && actor.Role != domain.RoleAdmin {
		return domain.Product{}, apperror.AccessDenied()
	}
	cleanProductDraft(&draft)
	if err := validateProductDraft(draft); err != nil {
		return domain.Product{}, err
	}

	product, err := s.queries.CreateProduct(ctx, actor.ID, draft)
	if err != nil {
		return domain.Product{}, apperror.Internal(err)
	}
	return product, nil
}

func (s *ProductService) Get(ctx context.Context, id int64) (domain.Product, error) {
	product, err := s.queries.ProductByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Product{}, apperror.ProductNotFound(id)
	}
	if err != nil {
		return domain.Product{}, apperror.Internal(err)
	}
	return product, nil
}

func (s *ProductService) List(ctx context.Context, filter domain.ProductFilter) (domain.ProductPage, error) {
	page, err := s.queries.ListProducts(ctx, filter)
	if err != nil {
		return domain.ProductPage{}, apperror.Internal(err)
	}
	return page, nil
}

func (s *ProductService) Update(ctx context.Context, actor requestctx.Actor, id int64, draft domain.ProductDraft) (domain.Product, error) {
	if actor.Role != domain.RoleSeller && actor.Role != domain.RoleAdmin {
		return domain.Product{}, apperror.AccessDenied()
	}

	current, err := s.Get(ctx, id)
	if err != nil {
		return domain.Product{}, err
	}
	if actor.Role == domain.RoleSeller && current.SellerID != actor.ID {
		return domain.Product{}, apperror.AccessDenied()
	}

	cleanProductDraft(&draft)
	if err := validateProductDraft(draft); err != nil {
		return domain.Product{}, err
	}
	product, err := s.queries.UpdateProduct(ctx, id, draft)
	if err != nil {
		return domain.Product{}, apperror.Internal(err)
	}
	return product, nil
}

func (s *ProductService) Archive(ctx context.Context, actor requestctx.Actor, id int64) error {
	if actor.Role != domain.RoleSeller && actor.Role != domain.RoleAdmin {
		return apperror.AccessDenied()
	}

	current, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if actor.Role == domain.RoleSeller && current.SellerID != actor.ID {
		return apperror.AccessDenied()
	}

	if err := s.queries.ArchiveProduct(ctx, id); err != nil {
		return apperror.Internal(err)
	}
	return nil
}

func cleanProductDraft(draft *domain.ProductDraft) {
	draft.Name = strings.TrimSpace(draft.Name)
	draft.Category = strings.TrimSpace(draft.Category)
	if draft.Description != nil {
		trimmed := strings.TrimSpace(*draft.Description)
		draft.Description = &trimmed
	}
}

func validateProductDraft(draft domain.ProductDraft) error {
	details := make(map[string]any)
	if draft.Name == "" {
		details["name"] = "must not be blank"
	}
	if draft.Category == "" {
		details["category"] = "must not be blank"
	}
	if draft.Price <= 0 {
		details["price"] = "must be greater than zero"
	}
	if draft.Stock < 0 {
		details["stock"] = "must be greater than or equal to zero"
	}
	switch draft.Status {
	case domain.ProductActive, domain.ProductInactive, domain.ProductArchived:
	default:
		details["status"] = "must be ACTIVE, INACTIVE or ARCHIVED"
	}
	if len(details) > 0 {
		return apperror.Validation(details)
	}
	return nil
}
