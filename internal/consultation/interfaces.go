package consultation

import (
	"context"

	"kubometr/internal/ai"
	"kubometr/internal/chat"
	"kubometr/internal/history"
	"kubometr/internal/state"
)

type aiCompleter interface {
	Complete(ctx context.Context, messages []ai.Message) (string, error)
}

type stateStore interface {
	Get(id chat.ID) state.UserState
	Set(id chat.ID, userState state.UserState)
	Delete(id chat.ID)
}

type historyStore interface {
	Save(ctx context.Context, userID int64, role string, text string) error
	LoadHistory(ctx context.Context, userID int64, limit int) ([]history.Message, error)
	Delete(ctx context.Context, userID int64) error
}

type userStore interface {
	GetOrCreate(ctx context.Context, id chat.ID) (int64, error)
}
