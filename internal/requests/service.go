// Package requests turns a consultation into a request for a manager: the
// client leaves a phone number, the manager gets the dialog and moves the
// request through its statuses, and the client can see them.
package requests

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"kubometr/internal/chat"
	"kubometr/internal/history"
)

type Status string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusReady      Status = "ready"
	StatusIssued     Status = "issued"
	StatusCancelled  Status = "cancelled"
)

var allStatuses = []Status{StatusNew, StatusInProgress, StatusReady, StatusIssued, StatusCancelled}

func (s Status) Title() string {
	switch s {
	case StatusInProgress:
		return "🔧 В работе"
	case StatusReady:
		return "📦 Готова"
	case StatusIssued:
		return "✅ Выдана"
	case StatusCancelled:
		return "❌ Отменена"
	}
	return "🆕 Принята"
}

// Next lists the statuses the manager can move a request to. An issued or
// cancelled request is closed.
func (s Status) Next() []Status {
	switch s {
	case StatusNew:
		return []Status{StatusInProgress, StatusReady, StatusCancelled}
	case StatusInProgress:
		return []Status{StatusReady, StatusCancelled}
	case StatusReady:
		return []Status{StatusIssued, StatusCancelled}
	}
	return nil
}

// previous lists the statuses a request can move to s from.
func (s Status) previous() []string {
	var from []string
	for _, status := range allStatuses {
		if slices.Contains(status.Next(), s) {
			from = append(from, string(status))
		}
	}
	return from
}

// clientText tells the client about a status change.
func (s Status) clientText(id int64) string {
	switch s {
	case StatusInProgress:
		return fmt.Sprintf("🔧 Заявка №%d в работе: менеджер собирает заказ.", id)
	case StatusReady:
		return fmt.Sprintf("📦 Заявка №%d готова, можно забирать!", id)
	case StatusIssued:
		return fmt.Sprintf("✅ Заявка №%d выдана. Спасибо, что выбрали Кубометр!", id)
	case StatusCancelled:
		return fmt.Sprintf("❌ Заявка №%d отменена. Если это ошибка, свяжитесь с менеджером или оформите новую заявку.", id)
	}
	return fmt.Sprintf("Заявка №%d: %s.", id, s.Title())
}

type Request struct {
	ID        int64
	Client    chat.ID
	Phone     string
	Question  string // the client's messages from the dialog
	Answer    string // the consultant's last answer
	Status    Status
	CreatedAt time.Time
}

const (
	historyLimit = 20
	listLimit    = 10
)

// moscow is used for request dates: the store works in Moscow time.
var moscow = time.FixedZone("MSK", 3*60*60)

type repository interface {
	Create(ctx context.Context, userID int64, req Request) (Request, error)
	List(ctx context.Context, userID int64, limit int) ([]Request, error)
	SetStatus(ctx context.Context, id int64, status Status) (Request, bool, error)
}

type userStore interface {
	GetOrCreate(ctx context.Context, id chat.ID) (int64, error)
}

type historyStore interface {
	LoadHistory(ctx context.Context, userID int64, limit int) ([]history.Message, error)
}

type Service struct {
	repo    repository
	users   userStore
	history historyStore

	// notifyManager delivers a new request to the manager; nil when no
	// manager chat is configured.
	notifyManager func(ctx context.Context, req Request) error
	// notifyClient sends a message to the client's chat in any messenger.
	notifyClient func(ctx context.Context, id chat.ID, text string) error
}

func NewService(
	repo repository,
	users userStore,
	history historyStore,
	notifyManager func(ctx context.Context, req Request) error,
	notifyClient func(ctx context.Context, id chat.ID, text string) error,
) *Service {
	return &Service{
		repo:          repo,
		users:         users,
		history:       history,
		notifyManager: notifyManager,
		notifyClient:  notifyClient,
	}
}

// Submit creates a request from the current dialog and returns the reply
// for the client.
func (s *Service) Submit(ctx context.Context, id chat.ID, phone string) (string, error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return "Не удалось прочитать номер телефона. Попробуйте ещё раз.", nil
	}

	userID, err := s.users.GetOrCreate(ctx, id)
	if err != nil {
		return "", fmt.Errorf("get or create user: %w", err)
	}

	messages, err := s.history.LoadHistory(ctx, userID, historyLimit)
	if err != nil {
		return "", fmt.Errorf("load history: %w", err)
	}

	req := fromDialog(messages)
	if req.Question == "" {
		return "Сначала опишите консультанту, что нужно сделать: заявка соберётся из вашего диалога.", nil
	}
	req.Client = id
	req.Phone = phone

	req, err = s.repo.Create(ctx, userID, req)
	if err != nil {
		return "", err
	}

	// The request is already saved, so a failed notification must not turn
	// into an error for the client: the manager still sees it in the database.
	if s.notifyManager == nil {
		slog.WarnContext(ctx, "manager chat is not configured, request only saved", "request_id", req.ID)
	} else if err := s.notifyManager(ctx, req); err != nil {
		slog.ErrorContext(ctx, "notify manager", "request_id", req.ID, "error", err)
	}

	return fmt.Sprintf("✅ Заявка №%d принята.\n\nМенеджер свяжется с вами по номеру %s. "+
		"Статус заявки можно посмотреть в разделе «🧾 Мои заявки».", req.ID, phone), nil
}

// fromDialog collects the client's messages and the consultant's last answer.
func fromDialog(messages []history.Message) Request {
	var questions []string
	var req Request
	for _, msg := range messages {
		switch msg.Role {
		case history.UserRole:
			questions = append(questions, "• "+msg.Text)
		case history.AIRole:
			req.Answer = msg.Text
		}
	}
	req.Question = strings.Join(questions, "\n")
	return req
}

// List returns the client's latest requests as a message.
func (s *Service) List(ctx context.Context, id chat.ID) (string, error) {
	userID, err := s.users.GetOrCreate(ctx, id)
	if err != nil {
		return "", fmt.Errorf("get or create user: %w", err)
	}

	list, err := s.repo.List(ctx, userID, listLimit)
	if err != nil {
		return "", err
	}
	if len(list) == 0 {
		return "У вас пока нет заявок.\n\nОпишите консультанту, что нужно сделать, " +
			"а потом нажмите «📝 Оформить заявку» — менеджер перезвонит.", nil
	}

	var b strings.Builder
	b.WriteString("🧾 Ваши заявки:")
	for _, req := range list {
		fmt.Fprintf(&b, "\n\n№%d от %s — %s\n%s",
			req.ID, req.CreatedAt.In(moscow).Format("02.01.2006"), req.Status.Title(), preview(req.Question))
	}
	return b.String(), nil
}

// preview shortens the client's first message for the request list.
func preview(question string) string {
	first, _, _ := strings.Cut(question, "\n")
	first = strings.TrimPrefix(first, "• ")
	if runes := []rune(first); len(runes) > 80 {
		return string(runes[:80]) + "…"
	}
	return first
}

// SetStatus moves a request to a new status and tells the client. ok is
// false when the request doesn't exist or can't move to this status, e.g.
// the button was pressed twice.
func (s *Service) SetStatus(ctx context.Context, id int64, status Status) (Request, bool, error) {
	req, ok, err := s.repo.SetStatus(ctx, id, status)
	if err != nil || !ok {
		return req, ok, err
	}

	if err := s.notifyClient(ctx, req.Client, status.clientText(req.ID)); err != nil {
		slog.ErrorContext(ctx, "notify client about request status", "request_id", req.ID, "error", err)
	}
	return req, true, nil
}

// ManagerText formats a request for the manager.
func (r Request) ManagerText() string {
	text := fmt.Sprintf("📝 Заявка №%d — %s\n\nТелефон: %s\nМессенджер: %s\n\nКлиент писал:\n%s",
		r.ID, r.Status.Title(), r.Phone, r.Client.Platform, r.Question)
	if r.Answer != "" {
		text += "\n\nКонсультант ответил:\n" + r.Answer
	}
	return text
}
