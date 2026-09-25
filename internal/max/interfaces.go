package max

import (
	"context"

	"kubometr/internal/chat"
	"kubometr/internal/requests"
)

type ConsultationService interface {
	Start(id chat.ID)
	Reset(ctx context.Context, id chat.ID) error
	Process(ctx context.Context, id chat.ID, question string) (string, error)
}

type RequestService interface {
	Submit(ctx context.Context, id chat.ID, phone string) (string, error)
	List(ctx context.Context, id chat.ID) (string, error)
	SetStatus(ctx context.Context, id int64, status requests.Status) (requests.Request, bool, error)
}

type MessageSender interface {
	SendMessage(ctx context.Context, chatID int64, text string, menu Menu) error
	SendTyping(ctx context.Context, chatID int64) error
	AnswerCallback(ctx context.Context, callbackID string, req *requests.Request, notification string) error
}
