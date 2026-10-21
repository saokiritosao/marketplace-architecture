package httpapi

import (
	"net/http"

	"marketplace/internal/api/generated"
	"marketplace/internal/apperror"
	"marketplace/internal/domain"
	"marketplace/internal/money"
)

func (s *Server) CreatePromoCode(w http.ResponseWriter, r *http.Request) {
	var body generated.PromoCodeCreate
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, r, err)
		return
	}

	discount, err := money.FromFloat(body.DiscountValue)
	if err != nil {
		writeError(w, r, apperror.Validation(map[string]any{"discount_value": "must have at most two decimal places"}))
		return
	}
	minimum, err := money.FromFloat(body.MinOrderAmount)
	if err != nil {
		writeError(w, r, apperror.Validation(map[string]any{"min_order_amount": "must have at most two decimal places"}))
		return
	}

	active := true
	if body.Active != nil {
		active = *body.Active
	}
	draft := domain.PromoDraft{
		Code:           body.Code,
		DiscountType:   domain.DiscountType(body.DiscountType),
		DiscountValue:  discount,
		MinOrderAmount: minimum,
		MaxUses:        body.MaxUses,
		ValidFrom:      body.ValidFrom,
		ValidUntil:     body.ValidUntil,
		Active:         active,
	}

	promo, err := s.promos.Create(r.Context(), mustActor(r.Context()), draft)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, promoResponse(promo))
}
