package history

import (
	"context"
	"fmt"

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

const saveQuery = `
	INSERT INTO messages (user_id, role, text)
	VALUES ($1, $2, $3)
	`

const loadHistoryQuery = `
	SELECT role, text, created_at
	FROM messages
	WHERE user_id = $1
	ORDER BY id DESC
	LIMIT $2
`

const deleteHistoryQuery = `
	DELETE FROM messages
	WHERE user_id = $1
`

func (r *Repository) Save(ctx context.Context, userID int64, role string, text string) error {
	_, err := r.pool.Exec(ctx, saveQuery, userID, role, text)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	return nil
}

func (r *Repository) LoadHistory(ctx context.Context, userID int64, limit int) ([]Message, error) {
	rows, err := r.pool.Query(ctx, loadHistoryQuery, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute load history query: %w", err)
	}
	defer rows.Close()

	var history []Message

	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.Role, &msg.Text, &msg.Time); err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}
		history = append(history, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}

	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}

	return history, nil
}

func (r *Repository) Delete(ctx context.Context, userID int64) error {
	if _, err := r.pool.Exec(ctx, deleteHistoryQuery, userID); err != nil {
		return fmt.Errorf("delete history: %w", err)
	}

	return nil
}
