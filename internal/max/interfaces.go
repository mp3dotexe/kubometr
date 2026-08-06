package max

import (
	"context"
)

type ConsultationProcessor interface {
	Process(ctx context.Context, chatID int64, question string) (string, error)
}