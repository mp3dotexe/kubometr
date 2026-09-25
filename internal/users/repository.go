package users

import (
	"context"
	"errors"
	"fmt"

	"kubometr/internal/chat"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool: pool,
	}
}

const selectUserQuery = `
	SELECT id FROM users
	WHERE platform = $1 AND external_id = $2
`

const insertUserQuery = `
	INSERT INTO users (platform, external_id)
	VALUES ($1, $2)
	ON CONFLICT (platform, external_id) DO NOTHING
	RETURNING id
`

// GetOrCreate returns the internal user ID for a chat, creating the user on
// first contact. It looks the user up first so that regular messages don't
// hit the unique constraint and burn sequence values.
func (r *Repository) GetOrCreate(ctx context.Context, id chat.ID) (int64, error) {
	userID, err := r.selectID(ctx, id)
	if err == nil {
		return userID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("select user id: %w", err)
	}

	err = r.pool.QueryRow(ctx, insertUserQuery, id.Platform, id.ChatID).Scan(&userID)
	if err == nil {
		return userID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("insert user: %w", err)
	}

	// A concurrent request created the user between our SELECT and INSERT.
	userID, err = r.selectID(ctx, id)
	if err != nil {
		return 0, fmt.Errorf("select user id after conflict: %w", err)
	}
	return userID, nil
}

func (r *Repository) selectID(ctx context.Context, id chat.ID) (int64, error) {
	var userID int64
	err := r.pool.QueryRow(ctx, selectUserQuery, id.Platform, id.ChatID).Scan(&userID)
	return userID, err
}
