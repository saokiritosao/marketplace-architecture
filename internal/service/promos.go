package service

import (
	"context"
	"net/http"
	"strings"

	"marketplace/internal/apperror"
	"marketplace/internal/domain"
	"marketplace/internal/requestctx"
	"marketplace/internal/store"
)

type PromoService struct {
	queries *store.Queries
}

func NewPromoService(queries *store.Queries) *PromoService {
	return &PromoService{queries: queries}
}

func (s *PromoService) Create(ctx context.Context, actor requestctx.Actor, draft domain.PromoDraft) (domain.PromoCode, error) {
	if actor.Role != domain.RoleSeller && actor.Role != domain.RoleAdmin {
		return domain.PromoCode{}, apperror.AccessDenied()
	}
	draft.Code = strings.ToUpper(strings.TrimSpace(draft.Code))
	if err := validatePromoDraft(draft); err != nil {
		return domain.PromoCode{}, err
	}
	promo, err := s.queries.CreatePromoCode(ctx, actor.ID, draft)
	if store.IsUniqueViolation(err) {
		return domain.PromoCode{}, apperror.New(http.StatusConflict, "PROMO_CODE_EXISTS", "Promo code already exists", nil)
	}
	if err != nil {
		return domain.PromoCode{}, apperror.Internal(err)
	}
	return promo, nil
}

func validatePromoDraft(draft domain.PromoDraft) error {
	details := make(map[string]any)
	if draft.Code == "" {
		details["code"] = "must not be blank"
	}
	if draft.DiscountValue <= 0 {
		details["discount_value"] = "must be greater than zero"
	}
	if draft.MinOrderAmount < 0 {
		details["min_order_amount"] = "must be greater than or equal to zero"
	}
	if draft.MaxUses <= 0 {
		details["max_uses"] = "must be greater than zero"
	}
	if !draft.ValidUntil.After(draft.ValidFrom) {
		details["valid_until"] = "must be later than valid_from"
	}
	switch draft.DiscountType {
	case domain.DiscountPercentage:
		if draft.DiscountValue > 10000 {
			details["discount_value"] = "percentage cannot be greater than 100"
		}
	case domain.DiscountFixed:
	default:
		details["discount_type"] = "must be PERCENTAGE or FIXED_AMOUNT"
	}
	if len(details) > 0 {
		return apperror.Validation(details)
	}
	return nil
}
