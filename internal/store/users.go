package store

import (
	"context"
	"strings"

	"marketplace/internal/domain"
)

func (q *Queries) CreateUser(ctx context.Context, email, passwordHash string, role domain.Role) (domain.User, error) {
	const query = `
		INSERT INTO users(email, password_hash, role)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, role, created_at`

	var user domain.User
	err := q.db.QueryRow(ctx, query, strings.ToLower(strings.TrimSpace(email)), passwordHash, role).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
	)
	return user, err
}

func (q *Queries) UserByEmail(ctx context.Context, email string) (domain.User, error) {
	const query = `
		SELECT id, email, password_hash, role, created_at
		FROM users
		WHERE email = $1`

	var user domain.User
	err := q.db.QueryRow(ctx, query, strings.ToLower(strings.TrimSpace(email))).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
	)
	return user, err
}

func (q *Queries) UserByID(ctx context.Context, id int64) (domain.User, error) {
	const query = `
		SELECT id, email, password_hash, role, created_at
		FROM users
		WHERE id = $1`

	var user domain.User
	err := q.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
	)
	return user, err
}

func (q *Queries) LockUser(ctx context.Context, id int64) error {
	const query = `SELECT id FROM users WHERE id = $1 FOR UPDATE`
	var lockedID int64
	return q.db.QueryRow(ctx, query, id).Scan(&lockedID)
}
