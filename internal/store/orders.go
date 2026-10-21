package store

import (
	"context"
	"time"

	"marketplace/internal/domain"
)

const orderColumns = `
	o.id,
	o.user_id,
	o.status,
	o.promo_code_id,
	p.code,
	ROUND(o.subtotal_amount * 100)::bigint,
	ROUND(o.discount_amount * 100)::bigint,
	ROUND(o.total_amount * 100)::bigint,
	o.created_at,
	o.updated_at`

func (q *Queries) HasActiveOrder(ctx context.Context, userID int64) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM orders
			WHERE user_id = $1
			  AND status IN ('CREATED', 'PAYMENT_PENDING')
		)`
	var exists bool
	err := q.db.QueryRow(ctx, query, userID).Scan(&exists)
	return exists, err
}

func (q *Queries) LastOperation(ctx context.Context, userID int64, operation domain.OperationType) (*time.Time, error) {
	const query = `
		SELECT created_at
		FROM user_operations
		WHERE user_id = $1 AND operation_type = $2
		ORDER BY created_at DESC
		LIMIT 1`

	var createdAt time.Time
	err := q.db.QueryRow(ctx, query, userID, operation).Scan(&createdAt)
	if err == ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &createdAt, nil
}

func (q *Queries) AddOperation(ctx context.Context, userID int64, operation domain.OperationType) error {
	const query = `INSERT INTO user_operations(user_id, operation_type) VALUES ($1, $2)`
	_, err := q.db.Exec(ctx, query, userID, operation)
	return err
}

func (q *Queries) CreateOrder(ctx context.Context, order domain.Order) (domain.Order, error) {
	const query = `
		INSERT INTO orders(
			user_id, status, promo_code_id, subtotal_amount, discount_amount, total_amount
		)
		VALUES ($1, $2, $3, $4::bigint / 100.0, $5::bigint / 100.0, $6::bigint / 100.0)
		RETURNING id, created_at, updated_at`

	err := q.db.QueryRow(ctx, query,
		order.UserID,
		order.Status,
		order.PromoCodeID,
		int64(order.SubtotalAmount),
		int64(order.DiscountAmount),
		int64(order.TotalAmount),
	).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
	return order, err
}

func (q *Queries) AddOrderItem(ctx context.Context, item domain.OrderItem) (domain.OrderItem, error) {
	const query = `
		INSERT INTO order_items(order_id, product_id, quantity, price_at_order)
		VALUES ($1, $2, $3, $4::bigint / 100.0)
		RETURNING id`

	err := q.db.QueryRow(ctx, query,
		item.OrderID,
		item.ProductID,
		item.Quantity,
		int64(item.PriceAtOrder),
	).Scan(&item.ID)
	return item, err
}

func (q *Queries) OrderByID(ctx context.Context, id int64) (domain.Order, error) {
	query := `
		SELECT ` + orderColumns + `
		FROM orders o
		LEFT JOIN promo_codes p ON p.id = o.promo_code_id
		WHERE o.id = $1`

	order, err := scanOrder(q.db.QueryRow(ctx, query, id))
	if err != nil {
		return domain.Order{}, err
	}
	order.Items, err = q.OrderItems(ctx, id)
	return order, err
}

func (q *Queries) LockOrderByID(ctx context.Context, id int64) (domain.Order, error) {
	query := `
		SELECT ` + orderColumns + `
		FROM orders o
		LEFT JOIN promo_codes p ON p.id = o.promo_code_id
		WHERE o.id = $1
		FOR UPDATE OF o`

	order, err := scanOrder(q.db.QueryRow(ctx, query, id))
	if err != nil {
		return domain.Order{}, err
	}
	order.Items, err = q.OrderItems(ctx, id)
	return order, err
}

func (q *Queries) OrderItems(ctx context.Context, orderID int64) ([]domain.OrderItem, error) {
	const query = `
		SELECT id, order_id, product_id, quantity, ROUND(price_at_order * 100)::bigint
		FROM order_items
		WHERE order_id = $1
		ORDER BY id`

	rows, err := q.db.Query(ctx, query, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.OrderItem, 0)
	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.ProductID,
			&item.Quantity,
			&item.PriceAtOrder,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (q *Queries) ReplaceOrderItems(ctx context.Context, orderID int64, items []domain.OrderItem) ([]domain.OrderItem, error) {
	if _, err := q.db.Exec(ctx, `DELETE FROM order_items WHERE order_id = $1`, orderID); err != nil {
		return nil, err
	}

	created := make([]domain.OrderItem, 0, len(items))
	for _, item := range items {
		item.OrderID = orderID
		stored, err := q.AddOrderItem(ctx, item)
		if err != nil {
			return nil, err
		}
		created = append(created, stored)
	}
	return created, nil
}

func (q *Queries) UpdateOrderAmounts(
	ctx context.Context,
	orderID int64,
	promoCodeID *int64,
	subtotal, discount, total int64,
) (time.Time, error) {
	const query = `
		UPDATE orders
		SET promo_code_id = $2,
		    subtotal_amount = $3::bigint / 100.0,
		    discount_amount = $4::bigint / 100.0,
		    total_amount = $5::bigint / 100.0
		WHERE id = $1
		RETURNING updated_at`

	var updatedAt time.Time
	err := q.db.QueryRow(ctx, query, orderID, promoCodeID, subtotal, discount, total).Scan(&updatedAt)
	return updatedAt, err
}

func (q *Queries) CancelOrder(ctx context.Context, orderID int64) (time.Time, error) {
	const query = `
		UPDATE orders
		SET status = 'CANCELED'
		WHERE id = $1
		RETURNING updated_at`

	var updatedAt time.Time
	err := q.db.QueryRow(ctx, query, orderID).Scan(&updatedAt)
	return updatedAt, err
}

type orderScanner interface {
	Scan(dest ...any) error
}

func scanOrder(row orderScanner) (domain.Order, error) {
	var order domain.Order
	err := row.Scan(
		&order.ID,
		&order.UserID,
		&order.Status,
		&order.PromoCodeID,
		&order.PromoCode,
		&order.SubtotalAmount,
		&order.DiscountAmount,
		&order.TotalAmount,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	return order, err
}

func (q *Queries) UpdateOrderStatus(ctx context.Context, orderID int64, status domain.OrderStatus) (time.Time, error) {
	const query = `
		UPDATE orders
		SET status = $2
		WHERE id = $1
		RETURNING updated_at`

	var updatedAt time.Time
	err := q.db.QueryRow(ctx, query, orderID, status).Scan(&updatedAt)
	return updatedAt, err
}
