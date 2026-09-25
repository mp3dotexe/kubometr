package max

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"kubometr/internal/chat"
)

const testSecret = "test-secret"

type mockConsultation struct {
	mu        sync.Mutex
	answer    string
	err       error
	started   []chat.ID
	reset     []chat.ID
	processed []string
}

func (m *mockConsultation) Start(id chat.ID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = append(m.started, id)
}

func (m *mockConsultation) Reset(_ context.Context, id chat.ID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reset = append(m.reset, id)
	return nil
}

func (m *mockConsultation) Process(_ context.Context, _ chat.ID, question string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processed = append(m.processed, question)
	return m.answer, m.err
}

type sentMessage struct {
	chatID int64
	text   string
}

type mockSender struct {
	mu     sync.Mutex
	sent   []sentMessage
	events []string
}

func (m *mockSender) SendTyping(_ context.Context, chatID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, "typing")
	return nil
}

func (m *mockSender) SendMessage(_ context.Context, chatID int64, text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, sentMessage{chatID: chatID, text: text})
	m.events = append(m.events, "message")
	return nil
}

func newTestHandler(consultation *mockConsultation) (*Handler, *mockSender) {
	sender := &mockSender{}
	return NewHandler(consultation, sender, testSecret, time.Second), sender
}

func serve(h *Handler, method, secret, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/max/webhook", strings.NewReader(body))
	if secret != "" {
		req.Header.Set("X-Max-Bot-Api-Secret", secret)
	}
	recorder := httptest.NewRecorder()
	h.HandleWebhook(recorder, req)
	h.Wait()
	return recorder
}

func messageUpdate(text string) string {
	return `{"update_type": "message_created", "message": {` +
		`"sender": {"user_id": 777, "name": "Иван"},` +
		`"recipient": {"chat_id": 12345, "chat_type": "dialog"},` +
		`"body": {"text": "` + text + `"}}}`
}

func TestHandleWebhook_Unauthorized(t *testing.T) {
	for _, secret := range []string{"", "wrong-secret"} {
		h, _ := newTestHandler(&mockConsultation{})

		recorder := serve(h, http.MethodPost, secret, messageUpdate("вопрос"))

		if recorder.Code != http.StatusUnauthorized {
			t.Errorf("secret %q: status = %d, want %d", secret, recorder.Code, http.StatusUnauthorized)
		}
	}
}

func TestHandleWebhook_MethodNotAllowed(t *testing.T) {
	h, _ := newTestHandler(&mockConsultation{})

	recorder := serve(h, http.MethodGet, testSecret, "")

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleWebhook_InvalidJSON(t *testing.T) {
	h, _ := newTestHandler(&mockConsultation{})

	recorder := serve(h, http.MethodPost, testSecret, "invalid json")

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestHandleWebhook_SendsAnswer(t *testing.T) {
	consultation := &mockConsultation{answer: "test answer"}
	h, sender := newTestHandler(consultation)

	recorder := serve(h, http.MethodPost, testSecret, messageUpdate("test question"))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if strings.Contains(recorder.Body.String(), "test answer") {
		t.Error("answer leaked into the webhook response instead of the send API")
	}

	want := chat.ID{Platform: chat.MAX, ChatID: 12345}
	if len(consultation.started) != 1 || consultation.started[0] != want {
		t.Errorf("started = %v, want [%v]", consultation.started, want)
	}
	if len(consultation.processed) != 1 || consultation.processed[0] != "test question" {
		t.Errorf("processed = %v", consultation.processed)
	}
	if len(sender.sent) != 1 || sender.sent[0] != (sentMessage{chatID: 12345, text: "test answer"}) {
		t.Errorf("sent = %v", sender.sent)
	}
}

func TestHandleWebhook_ConsultationErrorSendsFallback(t *testing.T) {
	h, sender := newTestHandler(&mockConsultation{err: errors.New("consultation error")})

	recorder := serve(h, http.MethodPost, testSecret, messageUpdate("test question"))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if len(sender.sent) != 1 || sender.sent[0].text != fallbackText {
		t.Errorf("sent = %v, want fallback message", sender.sent)
	}
}

func TestHandleWebhook_LongAnswerIsSplit(t *testing.T) {
	h, sender := newTestHandler(&mockConsultation{answer: strings.Repeat("я", maxMessageLimit+1)})

	serve(h, http.MethodPost, testSecret, messageUpdate("вопрос"))

	if len(sender.sent) != 2 {
		t.Fatalf("sent %d messages, want 2", len(sender.sent))
	}
}

func TestHandleWebhook_StartCommandResets(t *testing.T) {
	consultation := &mockConsultation{}
	h, sender := newTestHandler(consultation)

	serve(h, http.MethodPost, testSecret, messageUpdate("/start"))

	if len(consultation.reset) != 1 || len(consultation.processed) != 0 {
		t.Errorf("reset = %v, processed = %v", consultation.reset, consultation.processed)
	}
	if len(sender.sent) != 1 || sender.sent[0].text != welcomeText {
		t.Errorf("sent = %v, want welcome message", sender.sent)
	}
}

func TestHandleWebhook_BotStarted(t *testing.T) {
	consultation := &mockConsultation{}
	h, sender := newTestHandler(consultation)

	body := `{"update_type": "bot_started", "chat_id": 555, "user": {"user_id": 777}}`
	recorder := serve(h, http.MethodPost, testSecret, body)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	want := chat.ID{Platform: chat.MAX, ChatID: 555}
	if len(consultation.started) != 1 || consultation.started[0] != want {
		t.Errorf("started = %v, want [%v]", consultation.started, want)
	}
	if len(sender.sent) != 1 || sender.sent[0] != (sentMessage{chatID: 555, text: welcomeText}) {
		t.Errorf("sent = %v", sender.sent)
	}
}

func TestHandleWebhook_IgnoresIrrelevantUpdates(t *testing.T) {
	bodies := map[string]string{
		"no message":   `{"update_type": "message_created"}`,
		"no recipient": `{"update_type": "message_created", "message": {"body": {"text": "hi"}}}`,
		"from bot": `{"update_type": "message_created", "message": {"sender": {"user_id": 1, "is_bot": true},` +
			`"recipient": {"chat_id": 1}, "body": {"text": "hi"}}}`,
		"other type": `{"update_type": "message_removed"}`,
	}

	for name, body := range bodies {
		consultation := &mockConsultation{answer: "answer"}
		h, sender := newTestHandler(consultation)

		recorder := serve(h, http.MethodPost, testSecret, body)

		if recorder.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want %d", name, recorder.Code, http.StatusOK)
		}
		if len(consultation.processed) != 0 || len(sender.sent) != 0 {
			t.Errorf("%s: processed = %v, sent = %v", name, consultation.processed, sender.sent)
		}
	}
}

func TestHandleWebhook_TypingBeforeAnswer(t *testing.T) {
	h, sender := newTestHandler(&mockConsultation{answer: "test answer"})

	serve(h, http.MethodPost, testSecret, messageUpdate("вопрос"))

	if len(sender.events) < 2 || sender.events[0] != "typing" || sender.events[len(sender.events)-1] != "message" {
		t.Fatalf("events = %v, want typing first and the answer last", sender.events)
	}
}
