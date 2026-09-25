package max

import (
	"context"

	"kubometr/internal/chat"
)

type ConsultationProcessor interface {
	Process(ctx context.Context, id chat.ID, question string) (string, error)
}