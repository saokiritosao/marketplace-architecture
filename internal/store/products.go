package store

import (
	"context"

	"marketplace/internal/domain"
)

const productColumns = `
	id,
	seller_id,
	name,
	description,
	ROUND(price * 100)::bigint,
	stock,
	category,
	status,
	created_at,
	updated_at`

func (q *Queries) CreateProduct(ctx context.Context, sellerID int64, draft domain.ProductDraft) (domain.Product, error) {
	query := `
		INSERT INTO products(seller_id, name, description, price, stock, category, status)
		VALUES ($1, $2, $3, $4::bigint / 100.0, $5, $6, $7)
		RETURNING ` + productColumns

	row := q.db.QueryRow(ctx, query,
		sellerID,
		draft.Name,
		draft.Description,
		int64(draft.Price),
		draft.Stock,
		draft.Category,
		draft.Status,
	)
	return scanProduct(row)
}

func (q *Queries) ProductByID(ctx context.Context, id int64) (domain.Product, error) {
	query := `SELECT ` + productColumns + ` FROM products WHERE id = $1`
	return scanProduct(q.db.QueryRow(ctx, query, id))
}

func (q *Queries) ListProducts(ctx context.Context, filter domain.ProductFilter) (domain.ProductPage, error) {
	var status any
	if filter.Status != nil {
		status = string(*filter.Status)
	}
	var category any
	if filter.Category != nil {
		category = *filter.Category
	}

	const countQuery = `
		SELECT COUNT(*)
		FROM products
		WHERE ($1::product_status IS NULL OR status = $1)
		  AND ($2::text IS NULL OR category = $2)`

	page := domain.ProductPage{Page: filter.Page, Size: filter.Size, Items: make([]domain.Product, 0)}
	if err := q.db.QueryRow(ctx, countQuery, status, category).Scan(&page.Total); err != nil {
		return domain.ProductPage{}, err
	}

	query := `
		SELECT ` + productColumns + `
		FROM products
		WHERE ($1::product_status IS NULL OR status = $1)
		  AND ($2::text IS NULL OR category = $2)
		ORDER BY id
		LIMIT $3 OFFSET $4`

	rows, err := q.db.Query(ctx, query, status, category, filter.Size, filter.Page*filter.Size)
	if err != nil {
		return domain.ProductPage{}, err
	}
	defer rows.Close()

	for rows.Next() {
		product, err := scanProduct(rows)
		if err != nil {
			return domain.ProductPage{}, err
		}
		page.Items = append(page.Items, product)
	}
	if err := rows.Err(); err != nil {
		return domain.ProductPage{}, err
	}
	return page, nil
}

func (q *Queries) UpdateProduct(ctx context.Context, id int64, draft domain.ProductDraft) (domain.Product, error) {
	query := `
		UPDATE products
		SET name = $2,
		    description = $3,
		    price = $4::bigint / 100.0,
		    stock = $5,
		    category = $6,
		    status = $7
		WHERE id = $1
		RETURNING ` + productColumns

	return scanProduct(q.db.QueryRow(ctx, query,
		id,
		draft.Name,
		draft.Description,
		int64(draft.Price),
		draft.Stock,
		draft.Category,
		draft.Status,
	))
}

func (q *Queries) ArchiveProduct(ctx context.Context, id int64) error {
	const query = `UPDATE products SET status = 'ARCHIVED' WHERE id = $1`
	tag, err := q.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNoRows
	}
	return nil
}

func (q *Queries) LockProducts(ctx context.Context, ids []int64) (map[int64]domain.Product, error) {
	query := `
		SELECT ` + productColumns + `
		FROM products
		WHERE id = ANY($1)
		ORDER BY id
		FOR UPDATE`

	rows, err := q.db.Query(ctx, query, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make(map[int64]domain.Product, len(ids))
	for rows.Next() {
		var product domain.Product
		if err := rows.Scan(
			&product.ID,
			&product.SellerID,
			&product.Name,
			&product.Description,
			&product.Price,
			&product.Stock,
			&product.Category,
			&product.Status,
			&product.CreatedAt,
			&product.UpdatedAt,
		); err != nil {
			return nil, err
		}
		products[product.ID] = product
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return products, nil
}

func (q *Queries) SetProductStock(ctx context.Context, id int64, stock int) error {
	const query = `UPDATE products SET stock = $2 WHERE id = $1`
	_, err := q.db.Exec(ctx, query, id, stock)
	return err
}

type productScanner interface {
	Scan(dest ...any) error
}

func scanProduct(row productScanner) (domain.Product, error) {
	var product domain.Product
	err := row.Scan(
		&product.ID,
		&product.SellerID,
		&product.Name,
		&product.Description,
		&product.Price,
		&product.Stock,
		&product.Category,
		&product.Status,
		&product.CreatedAt,
		&product.UpdatedAt,
	)
	return product, err
}
