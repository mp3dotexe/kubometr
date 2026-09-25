package max

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"kubometr/internal/chat"
)

// maxMessageLimit is the maximum text length of one MAX message.
const maxMessageLimit = 4000

const (
	welcomeText = `Здравствуйте! Я — виртуальный консультант Кубометра.

Опишите, что вы хотите сделать, а я помогу подобрать материалы.

Например:

• Нужно утеплить балкон.
• Хочу сделать перегородку из гипсокартона.
• Нужна краска для ванной.
• Планирую залить стяжку пола.

Когда определитесь с материалами, нажмите «` + chat.ButtonSubmit + `» — менеджер перезвонит.`

	helpText = `📖 Опишите задачу, например «нужно утеплить балкон 6 м²», и я подскажу, какие материалы понадобятся.

` + chat.ButtonSubmit + ` — менеджер перезвонит и уточнит цены, наличие и доставку. В заявку попадёт ваш диалог с консультантом.
` + chat.ButtonRequests + ` — статусы ваших заявок.
` + chat.ButtonManager + ` — как связаться с менеджером.
` + chat.ButtonNewDialog + ` — начать заново, история очищается.

Команды: /new, /requests, /manager, /help`

	fallbackText = "Не удалось получить ответ от AI-консультанта. Попробуйте повторить вопрос чуть позже."
	errorText    = "Что-то пошло не так. Попробуйте ещё раз чуть позже."
)

type HandlerConfig struct {
	WebhookSecret string
	// ManagerChatID is the chat that receives requests; 0 when not set.
	ManagerChatID  int64
	ManagerContact string
	ProcessTimeout time.Duration
}

type Handler struct {
	consultation ConsultationService
	requests     RequestService
	sender       MessageSender
	cfg          HandlerConfig
	wg           sync.WaitGroup
}

func NewHandler(consultation ConsultationService, requests RequestService, sender MessageSender, cfg HandlerConfig) *Handler {
	return &Handler{
		consultation: consultation,
		requests:     requests,
		sender:       sender,
		cfg:          cfg,
	}
}

// HandleWebhook acknowledges the update right away and handles it in the
// background: an AI answer can take tens of seconds, while MAX expects a quick
// response and delivers the reply only through the send message API anyway.
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	receivedSecret := r.Header.Get("X-Max-Bot-Api-Secret")
	if subtle.ConstantTimeCompare([]byte(receivedSecret), []byte(h.cfg.WebhookSecret)) != 1 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	const maxBytes = 1024 * 1024
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	var update Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	h.wg.Add(1)
	go func() {
		defer h.wg.Done()

		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), h.cfg.ProcessTimeout)
		defer cancel()
		h.handleUpdate(ctx, update)
	}()

	w.WriteHeader(http.StatusOK)
}

// Wait blocks until all updates accepted so far are handled.
func (h *Handler) Wait() {
	h.wg.Wait()
}

func (h *Handler) handleUpdate(ctx context.Context, update Update) {
	switch update.UpdateType {
	case updateBotStarted:
		if update.ChatID != 0 {
			h.restart(ctx, chat.ID{Platform: chat.MAX, ChatID: update.ChatID})
		}

	case updateMessageCreated:
		msg := update.Message
		if msg == nil || msg.Body == nil || msg.Recipient == nil || msg.Recipient.ChatID == 0 {
			return
		}
		if msg.Sender != nil && msg.Sender.IsBot {
			return
		}
		id := chat.ID{Platform: chat.MAX, ChatID: msg.Recipient.ChatID}
		if phone := msg.Body.ContactPhone(); phone != "" {
			h.submit(ctx, id, phone)
			return
		}
		h.handleMessage(ctx, id, msg.Body.Text)

	case updateMessageCallback:
		h.handleCallback(ctx, update)
	}
}

func (h *Handler) handleMessage(ctx context.Context, id chat.ID, text string) {
	switch strings.TrimSpace(text) {
	case "/start", "/new", chat.ButtonNewDialog:
		h.restart(ctx, id)
		return
	case "/help", chat.ButtonHelp:
		h.send(ctx, id, helpText, FullMenu)
		return
	case "/requests", chat.ButtonRequests:
		list, err := h.requests.List(ctx, id)
		if err != nil {
			slog.ErrorContext(ctx, "list requests", "platform", id.Platform, "chat_id", id.ChatID, "error", err)
			list = errorText
		}
		h.send(ctx, id, list, NoMenu)
		return
	case "/manager", chat.ButtonManager:
		h.send(ctx, id, chat.ManagerText(h.cfg.ManagerContact), NoMenu)
		return
	case "/id":
		// Helps to find the chat ID for MANAGER_CHAT_ID.
		h.send(ctx, id, fmt.Sprintf("ID этого чата: %d", id.ChatID), NoMenu)
		return
	}

	// The manager chat only receives requests: its messages aren't questions.
	if id.ChatID == h.cfg.ManagerChatID {
		return
	}

	// MAX has no "Consultation" button, so every message goes straight to the
	// consultant. This also covers chats whose state was lost on restart.
	h.consultation.Start(id)

	stopTyping := chat.KeepTyping(ctx, chat.TypingInterval, func(ctx context.Context) error {
		return h.sender.SendTyping(ctx, id.ChatID)
	})
	answer, err := h.consultation.Process(ctx, id, text)
	stopTyping()
	if err != nil {
		slog.ErrorContext(ctx, "process consultation", "platform", id.Platform, "chat_id", id.ChatID, "error", err)
		h.send(ctx, id, fallbackText, NoMenu)
		return
	}

	h.send(ctx, id, answer, AnswerMenu)
}

func (h *Handler) submit(ctx context.Context, id chat.ID, phone string) {
	reply, err := h.requests.Submit(ctx, id, phone)
	if err != nil {
		slog.ErrorContext(ctx, "submit request", "platform", id.Platform, "chat_id", id.ChatID, "error", err)
		reply = errorText
	}
	h.send(ctx, id, reply, NoMenu)
}

// handleCallback applies a status button pressed by the manager.
func (h *Handler) handleCallback(ctx context.Context, update Update) {
	cb := update.Callback
	if cb == nil || cb.CallbackID == "" {
		return
	}
	// Status buttons are only sent to the manager chat, so a press from any
	// other chat is not the manager's.
	msg := update.Message
	if h.cfg.ManagerChatID == 0 || msg == nil || msg.Recipient == nil || msg.Recipient.ChatID != h.cfg.ManagerChatID {
		return
	}
	requestID, status, ok := parseStatusPayload(cb.Payload)
	if !ok {
		return
	}

	req, changed, err := h.requests.SetStatus(ctx, requestID, status)
	switch {
	case err != nil:
		slog.ErrorContext(ctx, "set request status", "request_id", requestID, "error", err)
		err = h.sender.AnswerCallback(ctx, cb.CallbackID, nil, "Не удалось изменить статус, попробуйте ещё раз")
	case !changed:
		err = h.sender.AnswerCallback(ctx, cb.CallbackID, nil, "Статус заявки уже изменён")
	default:
		err = h.sender.AnswerCallback(ctx, cb.CallbackID, &req, fmt.Sprintf("Заявка №%d: %s", req.ID, req.Status.Title()))
	}
	if err != nil {
		slog.ErrorContext(ctx, "answer max callback", "request_id", requestID, "error", err)
	}
}

func (h *Handler) restart(ctx context.Context, id chat.ID) {
	if err := h.consultation.Reset(ctx, id); err != nil {
		slog.ErrorContext(ctx, "reset consultation", "platform", id.Platform, "chat_id", id.ChatID, "error", err)
	}
	h.consultation.Start(id)
	h.send(ctx, id, welcomeText, FullMenu)
}

// send splits a long text into several messages; the menu goes under the
// last one.
func (h *Handler) send(ctx context.Context, id chat.ID, text string, menu Menu) {
	parts := chat.SplitText(text, maxMessageLimit)
	for i, part := range parts {
		partMenu := NoMenu
		if i == len(parts)-1 {
			partMenu = menu
		}
		if err := h.sender.SendMessage(ctx, id.ChatID, part, partMenu); err != nil {
			slog.ErrorContext(ctx, "send max message", "chat_id", id.ChatID, "error", err)
			return
		}
	}
}
