// Package requests turns a consultation into a request for a manager: the
// client leaves a phone number, the manager gets what the client decided to
// buy and moves the request through its statuses, and the client sees them.
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
	Items     string // the consultant's final list the client agreed to
	Question  string // the client's messages from the dialog
	Answer    string // the consultant's last answer
	Status    Status
	CreatedAt time.Time
}

const (
	historyLimit = 20
	listLimit    = 10
)

// orderHeader starts the consultant's final list, see the consultant prompt.
const orderHeader = "Итого к заказу"

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

	reply := fmt.Sprintf("✅ Заявка №%d принята.", req.ID)
	if req.Items != "" {
		reply += "\n\n" + req.Items
	}
	return reply + fmt.Sprintf("\n\nМенеджер позвонит вам по номеру %s, уточнит цены и наличие. "+
		"Статус заявки — в разделе «🧾 Мои заявки».", phone), nil
}

// fromDialog collects the client's messages, the consultant's last answer
// and the latest final list the client agreed to.
func fromDialog(messages []history.Message) Request {
	var questions []string
	var req Request
	for _, msg := range messages {
		switch msg.Role {
		case history.UserRole:
			questions = append(questions, "• "+msg.Text)
		case history.AIRole:
			req.Answer = msg.Text
			if order := orderFrom(msg.Text); order != "" {
				req.Items = order
			}
		}
	}
	req.Question = strings.Join(questions, "\n")
	return req
}

// orderFrom cuts the final list out of a consultant answer: the header line
// and the list items right after it. It returns "" when there is none.
func orderFrom(answer string) string {
	start := strings.LastIndex(answer, orderHeader)
	if start < 0 {
		return ""
	}

	lines := strings.Split(answer[start:], "\n")
	order := []string{strings.TrimSpace(lines[0])}
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "•") || strings.HasPrefix(line, "-") {
			order = append(order, line)
		} else if line != "" || len(order) > 1 {
			break // the text after the list
		}
	}
	if len(order) == 1 {
		return ""
	}
	return strings.Join(order, "\n")
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
		summary := req.Items
		if summary == "" {
			summary = req.Question
		}
		fmt.Fprintf(&b, "\n\n№%d от %s — %s\n%s",
			req.ID, req.CreatedAt.In(moscow).Format("02.01.2006"), req.Status.Title(), preview(summary))
	}
	return b.String(), nil
}

// preview shortens the task from the list header ("Итого к заказу — стяжка
// пола 15 м²:"), or the first line of the list or question, for the request
// list.
func preview(summary string) string {
	first, rest, _ := strings.Cut(summary, "\n")
	if task, ok := strings.CutPrefix(first, orderHeader); ok {
		first = strings.Trim(task, " —-:")
		if first == "" {
			first, _, _ = strings.Cut(rest, "\n")
		}
	}
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

// ManagerText formats a request for the manager: the task and the items
// first, the dialog below for context.
func (r Request) ManagerText() string {
	text := fmt.Sprintf("📝 Заявка №%d — %s\n\nТелефон: %s\nМессенджер: %s",
		r.ID, r.Status.Title(), r.Phone, r.Client.Platform)
	if r.Items != "" {
		text += "\n\n" + r.Items
	}
	text += "\n\nКлиент писал:\n" + r.Question
	// The items already sum up the consultant's answer; without them it is
	// the manager's best clue to what the client wants.
	if r.Items == "" && r.Answer != "" {
		text += "\n\nКонсультант ответил:\n" + r.Answer
	}
	return text
}
