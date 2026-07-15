package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
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

func (r *Repository) GetOrCreate(ctx context.Context, telegramID int64) (int64, error) {
	var userID int64

	insertQuery := `
	INSERT INTO users (telegram_id)
	VALUES ($1)
	RETURNING id;
	`
	err := r.pool.QueryRow(ctx, insertQuery, telegramID).Scan(&userID)

	if err == nil {
		return userID, nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		selectQuery := `SELECT id FROM users WHERE telegram_id = $1;`
		err = r.pool.QueryRow(ctx, selectQuery, telegramID).Scan(&userID)
		if err != nil {
			return 0, fmt.Errorf("select user id: %w", err)
		}
		return userID, nil
	}
	return 0, fmt.Errorf("insert user: %w", err)
}
