package max

import (
	"context"
	"crypto/subtle"
	"encoding/json"
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
• Планирую залить стяжку пола.`

	helpText = `📖 Просто опишите задачу, и я подскажу, какие материалы понадобятся.

/start — начать новый диалог (история очищается)
/help — показать помощь`

	fallbackText = "Не удалось получить ответ от AI-консультанта. Попробуйте повторить вопрос чуть позже."
)

type Handler struct {
	consultation   ConsultationService
	sender         MessageSender
	webhookSecret  string
	processTimeout time.Duration
	wg             sync.WaitGroup
}

func NewHandler(consultation ConsultationService, sender MessageSender, webhookSecret string, processTimeout time.Duration) *Handler {
	return &Handler{
		consultation:   consultation,
		sender:         sender,
		webhookSecret:  webhookSecret,
		processTimeout: processTimeout,
	}
}

// HandleWebhook acknowledges the update right away and handles it in the
// background: an AI answer can take tens of seconds, while MAX expects a quick
// response and delivers the reply only through the send message API anyway.
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	receivedSecret := r.Header.Get("X-Max-Bot-Api-Secret")
	if subtle.ConstantTimeCompare([]byte(receivedSecret), []byte(h.webhookSecret)) != 1 {
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

		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), h.processTimeout)
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
		h.handleMessage(ctx, chat.ID{Platform: chat.MAX, ChatID: msg.Recipient.ChatID}, msg.Body.Text)
	}
}

func (h *Handler) handleMessage(ctx context.Context, id chat.ID, text string) {
	switch strings.TrimSpace(text) {
	case "/start":
		h.restart(ctx, id)
		return
	case "/help":
		h.send(ctx, id, helpText)
		return
	}

	// MAX has no "Consultation" button, so every message goes straight to the
	// consultant. This also covers chats whose state was lost on restart.
	h.consultation.Start(id)

	answer, err := h.consultation.Process(ctx, id, text)
	if err != nil {
		slog.ErrorContext(ctx, "process consultation", "platform", id.Platform, "chat_id", id.ChatID, "error", err)
		h.send(ctx, id, fallbackText)
		return
	}

	h.send(ctx, id, answer)
}

func (h *Handler) restart(ctx context.Context, id chat.ID) {
	if err := h.consultation.Reset(ctx, id); err != nil {
		slog.ErrorContext(ctx, "reset consultation", "platform", id.Platform, "chat_id", id.ChatID, "error", err)
	}
	h.consultation.Start(id)
	h.send(ctx, id, welcomeText)
}

func (h *Handler) send(ctx context.Context, id chat.ID, text string) {
	for _, part := range chat.SplitText(text, maxMessageLimit) {
		if err := h.sender.SendMessage(ctx, id.ChatID, part); err != nil {
			slog.ErrorContext(ctx, "send max message", "chat_id", id.ChatID, "error", err)
			return
		}
	}
}
