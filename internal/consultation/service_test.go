package consultation

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"kubometr/internal/ai"
	"kubometr/internal/chat"
	"kubometr/internal/history"
	"kubometr/internal/state"
)

var testChat = chat.ID{Platform: chat.Telegram, ChatID: 1}

type fakeAI struct {
	answer   string
	err      error
	calls    int
	received []ai.Message
}

func (f *fakeAI) Complete(_ context.Context, messages []ai.Message) (string, error) {
	f.calls++
	f.received = messages
	return f.answer, f.err
}

type fakeHistory struct {
	messages map[int64][]history.Message
	saveErr  error
}

func newFakeHistory() *fakeHistory {
	return &fakeHistory{messages: make(map[int64][]history.Message)}
}

func (f *fakeHistory) Save(_ context.Context, userID int64, role string, text string) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.messages[userID] = append(f.messages[userID], history.Message{Role: history.Role(role), Text: text})
	return nil
}

func (f *fakeHistory) LoadHistory(_ context.Context, userID int64, limit int) ([]history.Message, error) {
	msgs := f.messages[userID]
	if len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
	}
	return msgs, nil
}

func (f *fakeHistory) Delete(_ context.Context, userID int64) error {
	delete(f.messages, userID)
	return nil
}

type fakeUsers struct {
	ids map[chat.ID]int64
}

func (f *fakeUsers) GetOrCreate(_ context.Context, id chat.ID) (int64, error) {
	if f.ids == nil {
		f.ids = make(map[chat.ID]int64)
	}
	if userID, ok := f.ids[id]; ok {
		return userID, nil
	}
	f.ids[id] = int64(len(f.ids) + 1)
	return f.ids[id], nil
}

type testDeps struct {
	service *Service
	ai      *fakeAI
	history *fakeHistory
	users   *fakeUsers
}

func newTestService(answer string) testDeps {
	d := testDeps{
		ai:      &fakeAI{answer: answer},
		history: newFakeHistory(),
		users:   &fakeUsers{},
	}
	d.service = New(state.New(), d.ai, time.Second, 3*time.Second, 20, 1, d.history, d.users)
	return d
}

func TestProcessIdleAsksToStart(t *testing.T) {
	d := newTestService("ответ")

	answer, err := d.service.Process(context.Background(), testChat, "вопрос")
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if !strings.Contains(answer, "Консультация") {
		t.Fatalf("answer = %q, want hint about the consultation button", answer)
	}
	if d.ai.calls != 0 {
		t.Fatalf("AI called %d times in idle state", d.ai.calls)
	}
}

func TestProcessRejectsBadInput(t *testing.T) {
	cases := []struct {
		name     string
		question string
	}{
		{name: "empty", question: "   "},
		{name: "too long", question: strings.Repeat("я", 21)},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := newTestService("ответ")
			d.service.Start(testChat)

			if _, err := d.service.Process(context.Background(), testChat, c.question); err != nil {
				t.Fatalf("Process() error = %v", err)
			}
			if d.ai.calls != 0 {
				t.Fatalf("AI called %d times, want 0", d.ai.calls)
			}
		})
	}
}

func TestProcessSavesDialog(t *testing.T) {
	d := newTestService("  Нужна цементно-песчаная смесь.  ")
	d.service.Start(testChat)

	answer, err := d.service.Process(context.Background(), testChat, " Хочу залить стяжку ")
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if answer != "Нужна цементно-песчаная смесь." {
		t.Fatalf("answer = %q", answer)
	}

	saved := d.history.messages[1]
	if len(saved) != 2 {
		t.Fatalf("saved %d messages, want 2", len(saved))
	}
	if saved[0].Role != history.UserRole || saved[0].Text != "Хочу залить стяжку" {
		t.Fatalf("user message = %+v", saved[0])
	}
	if saved[1].Role != history.AIRole || saved[1].Text != "Нужна цементно-песчаная смесь." {
		t.Fatalf("ai message = %+v", saved[1])
	}

	if len(d.ai.received) != 2 || d.ai.received[0].Role != ai.RoleSystem || d.ai.received[1].Role != ai.RoleUser {
		t.Fatalf("AI received %+v, want system + user messages", d.ai.received)
	}
}

func TestProcessRateLimit(t *testing.T) {
	d := newTestService("ответ")
	d.service.Start(testChat)

	if _, err := d.service.Process(context.Background(), testChat, "первый"); err != nil {
		t.Fatalf("first Process() error = %v", err)
	}
	answer, err := d.service.Process(context.Background(), testChat, "второй")
	if err != nil {
		t.Fatalf("second Process() error = %v", err)
	}
	if !strings.Contains(answer, "подождите") {
		t.Fatalf("answer = %q, want rate limit message", answer)
	}
	if d.ai.calls != 1 {
		t.Fatalf("AI called %d times, want 1", d.ai.calls)
	}
}

func TestProcessBusy(t *testing.T) {
	d := newTestService("ответ")
	d.service.Start(testChat)
	d.service.aiLimiter <- struct{}{}

	answer, err := d.service.Process(context.Background(), testChat, "вопрос")
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if !strings.Contains(answer, "много запросов") {
		t.Fatalf("answer = %q, want busy message", answer)
	}
	if d.users.ids != nil {
		t.Fatal("user created although request was rejected")
	}
}

func TestProcessAIError(t *testing.T) {
	d := newTestService("")
	d.ai.err = errors.New("boom")
	d.service.Start(testChat)

	if _, err := d.service.Process(context.Background(), testChat, "вопрос"); err == nil {
		t.Fatal("Process() error = nil, want error")
	}
}

func TestProcessEmptyAnswerNotSaved(t *testing.T) {
	d := newTestService("   ")
	d.service.Start(testChat)

	answer, err := d.service.Process(context.Background(), testChat, "вопрос")
	if err != nil {
		t.Fatalf("Process() error = %v", err)
	}
	if !strings.Contains(answer, "пустой ответ") {
		t.Fatalf("answer = %q", answer)
	}
	if got := len(d.history.messages[1]); got != 1 {
		t.Fatalf("saved %d messages, want only the user one", got)
	}
}

func TestResetClearsStateAndHistory(t *testing.T) {
	d := newTestService("ответ")
	d.service.Start(testChat)
	if _, err := d.service.Process(context.Background(), testChat, "вопрос"); err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if err := d.service.Reset(context.Background(), testChat); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}

	if len(d.history.messages[1]) != 0 {
		t.Fatal("history not cleared")
	}
	answer, _ := d.service.Process(context.Background(), testChat, "вопрос")
	if !strings.Contains(answer, "Консультация") {
		t.Fatalf("after Reset answer = %q, want idle hint", answer)
	}
}

func TestPlatformsDoNotShareUsers(t *testing.T) {
	d := newTestService("ответ")
	tg := chat.ID{Platform: chat.Telegram, ChatID: 7}
	mx := chat.ID{Platform: chat.MAX, ChatID: 7}
	d.service.Start(tg)
	d.service.Start(mx)

	if _, err := d.service.Process(context.Background(), tg, "из телеграма"); err != nil {
		t.Fatalf("Process(tg) error = %v", err)
	}
	if _, err := d.service.Process(context.Background(), mx, "из MAX"); err != nil {
		t.Fatalf("Process(max) error = %v", err)
	}

	if d.users.ids[tg] == d.users.ids[mx] {
		t.Fatal("telegram and MAX chats with the same ID share a user")
	}
	// The MAX request must not see the Telegram dialog.
	for _, m := range d.ai.received {
		if m.Content == "из телеграма" {
			t.Fatal("MAX request contains Telegram history")
		}
	}
}

func TestCanAskAI_RateLimit(t *testing.T) {
	s := &Service{
		aiRateLimit:   3 * time.Second,
		lastAIRequest: make(map[chat.ID]time.Time),
	}

	now := time.Now()
	first := chat.ID{Platform: chat.Telegram, ChatID: 1}
	second := chat.ID{Platform: chat.Telegram, ChatID: 2}

	if !s.canAskAI(first, now) {
		t.Errorf("expected true on first call, got false")
	}
	if s.canAskAI(first, now) {
		t.Errorf("expected false on second call, got true")
	}
	if !s.canAskAI(second, now) {
		t.Errorf("expected true for another chat, got false")
	}
	if !s.canAskAI(first, now.Add(4*time.Second)) {
		t.Errorf("expected true after the window, got false")
	}
}

func TestCanAskAI_PrunesOldEntries(t *testing.T) {
	s := &Service{
		aiRateLimit:   time.Second,
		lastAIRequest: make(map[chat.ID]time.Time),
	}

	old := time.Now()
	for i := range rateLimitPruneThreshold {
		s.lastAIRequest[chat.ID{Platform: chat.Telegram, ChatID: int64(i)}] = old
	}

	s.canAskAI(chat.ID{Platform: chat.MAX, ChatID: 1}, old.Add(time.Minute))

	if len(s.lastAIRequest) != 1 {
		t.Fatalf("map size = %d, want 1 after pruning", len(s.lastAIRequest))
	}
}
