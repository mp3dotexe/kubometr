package consultation

import (
	"context"
	"kubometr/internal/state"
	"kubometr/internal/history"
)

type aiAsker interface {
	Ask(ctx context.Context, prompt string) (string, error)
}

type stateStore interface {
	Get(chatID int64) state.UserState
	Set(chatID int64, userState state.UserState)
	Delete(chatID int64) 
}

type historyStore interface {
	Save(ctx context.Context, userID int64, role string, text string) error
	LoadHistory(ctx context.Context, userID int64, limit int) ([]history.Message, error)
	Delete(ctx context.Context, userID int64) error
}

type userStore interface {
	GetOrCreate(ctx context.Context, chatID int64) (int64, error)
}