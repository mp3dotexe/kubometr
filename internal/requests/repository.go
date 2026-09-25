package requests

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool: pool,
	}
}

const createQuery = `
	INSERT INTO requests (user_id, phone, question, answer)
	VALUES ($1, $2, $3, $4)
	RETURNING id, status, created_at
`

const listQuery = `
	SELECT id, phone, question, answer, status, created_at
	FROM requests
	WHERE user_id = $1
	ORDER BY id DESC
	LIMIT $2
`

// setStatusQuery changes the status only when it differs, so a repeated
// button press doesn't notify the client twice.
const setStatusQuery = `
	UPDATE requests r
	SET status = $2
	FROM users u
	WHERE r.id = $1 AND r.status <> $2 AND u.id = r.user_id
	RETURNING r.id, r.phone, r.question, r.answer, r.status, r.created_at, u.platform, u.external_id
`

func (r *Repository) Create(ctx context.Context, userID int64, req Request) (Request, error) {
	err := r.pool.QueryRow(ctx, createQuery, userID, req.Phone, req.Question, req.Answer).
		Scan(&req.ID, &req.Status, &req.CreatedAt)
	if err != nil {
		return Request{}, fmt.Errorf("insert request: %w", err)
	}
	return req, nil
}

func (r *Repository) List(ctx context.Context, userID int64, limit int) ([]Request, error) {
	rows, err := r.pool.Query(ctx, listQuery, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("select requests: %w", err)
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Request, error) {
		var req Request
		err := row.Scan(&req.ID, &req.Phone, &req.Question, &req.Answer, &req.Status, &req.CreatedAt)
		return req, err
	})
}

// SetStatus returns the updated request with its client chat, or ok = false
// when the request doesn't exist or already has this status.
func (r *Repository) SetStatus(ctx context.Context, id int64, status Status) (req Request, ok bool, err error) {
	err = r.pool.QueryRow(ctx, setStatusQuery, id, status).Scan(
		&req.ID, &req.Phone, &req.Question, &req.Answer, &req.Status, &req.CreatedAt,
		&req.Client.Platform, &req.Client.ChatID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Request{}, false, nil
	}
	if err != nil {
		return Request{}, false, fmt.Errorf("update request status: %w", err)
	}
	return req, true, nil
}
