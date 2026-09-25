package max

import (
	"context"

	"kubometr/internal/chat"
)

type ConsultationService interface {
	Start(id chat.ID)
	Reset(ctx context.Context, id chat.ID) error
	Process(ctx context.Context, id chat.ID, question string) (string, error)
}

type MessageSender interface {
	SendMessage(ctx context.Context, chatID int64, text string) error
}
