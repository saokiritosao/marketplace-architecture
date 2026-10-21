package store

import (
	"context"

	"marketplace/internal/domain"
)

const promoColumns = `
	id,
	seller_id,
	code,
	discount_type,
	ROUND(discount_value * 100)::bigint,
	ROUND(min_order_amount * 100)::bigint,
	max_uses,
	current_uses,
	valid_from,
	valid_until,
	active,
	created_at`

func (q *Queries) CreatePromoCode(ctx context.Context, sellerID int64, draft domain.PromoDraft) (domain.PromoCode, error) {
	query := `
		INSERT INTO promo_codes(
			seller_id, code, discount_type, discount_value, min_order_amount,
			max_uses, valid_from, valid_until, active
		)
		VALUES ($1, $2, $3, $4::bigint / 100.0, $5::bigint / 100.0, $6, $7, $8, $9)
		RETURNING ` + promoColumns

	return scanPromo(q.db.QueryRow(ctx, query,
		sellerID,
		draft.Code,
		draft.DiscountType,
		int64(draft.DiscountValue),
		int64(draft.MinOrderAmount),
		draft.MaxUses,
		draft.ValidFrom,
		draft.ValidUntil,
		draft.Active,
	))
}

func (q *Queries) LockPromoByCode(ctx context.Context, code string) (domain.PromoCode, error) {
	query := `SELECT ` + promoColumns + ` FROM promo_codes WHERE code = $1 FOR UPDATE`
	return scanPromo(q.db.QueryRow(ctx, query, code))
}

func (q *Queries) LockPromoByID(ctx context.Context, id int64) (domain.PromoCode, error) {
	query := `SELECT ` + promoColumns + ` FROM promo_codes WHERE id = $1 FOR UPDATE`
	return scanPromo(q.db.QueryRow(ctx, query, id))
}

func (q *Queries) ChangePromoUses(ctx context.Context, id int64, delta int) error {
	const query = `
		UPDATE promo_codes
		SET current_uses = GREATEST(0, current_uses + $2)
		WHERE id = $1`
	_, err := q.db.Exec(ctx, query, id, delta)
	return err
}

type promoScanner interface {
	Scan(dest ...any) error
}

func scanPromo(row promoScanner) (domain.PromoCode, error) {
	var promo domain.PromoCode
	err := row.Scan(
		&promo.ID,
		&promo.SellerID,
		&promo.Code,
		&promo.DiscountType,
		&promo.DiscountValue,
		&promo.MinOrderAmount,
		&promo.MaxUses,
		&promo.CurrentUses,
		&promo.ValidFrom,
		&promo.ValidUntil,
		&promo.Active,
		&promo.CreatedAt,
	)
	return promo, err
}
