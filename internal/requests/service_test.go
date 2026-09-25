package requests

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"kubometr/internal/chat"
	"kubometr/internal/history"
)

var client = chat.ID{Platform: chat.MAX, ChatID: 42}

type fakeRepo struct {
	created []Request
}

func (f *fakeRepo) Create(_ context.Context, _ int64, req Request) (Request, error) {
	req.ID = int64(len(f.created) + 1)
	req.Status = StatusNew
	req.CreatedAt = time.Date(2026, 9, 25, 22, 30, 0, 0, time.UTC)
	f.created = append(f.created, req)
	return req, nil
}

func (f *fakeRepo) List(context.Context, int64, int) ([]Request, error) {
	return f.created, nil
}

func (f *fakeRepo) SetStatus(_ context.Context, id int64, status Status) (Request, bool, error) {
	for i := range f.created {
		if f.created[i].ID == id && slices.Contains(status.previous(), string(f.created[i].Status)) {
			f.created[i].Status = status
			return f.created[i], true, nil
		}
	}
	return Request{}, false, nil
}

type fakeUsers struct{}

func (fakeUsers) GetOrCreate(context.Context, chat.ID) (int64, error) { return 1, nil }

type fakeHistory []history.Message

func (f fakeHistory) LoadHistory(context.Context, int64, int) ([]history.Message, error) {
	return f, nil
}

type sent struct {
	to   chat.ID
	text string
}

const testItems = "Итого к заказу — утепление балкона 6 м²:\n• Технониколь, ЭППС 50 мм или аналог — 2 упаковки"

func newTestService(dialog fakeHistory, managerErr error) (*Service, *fakeRepo, *[]Request, *[]sent) {
	repo := &fakeRepo{}
	var toManager []Request
	var toClient []sent
	s := NewService(repo, fakeUsers{}, dialog,
		func(_ context.Context, req Request) error {
			toManager = append(toManager, req)
			return managerErr
		},
		func(_ context.Context, id chat.ID, text string) error {
			toClient = append(toClient, sent{id, text})
			return nil
		},
	)
	return s, repo, &toManager, &toClient
}

const finalAnswer = "Для балкона лучше ЭППС: не боится влаги.\n\n" + testItems + "  \n\n" +
	"Если всё подходит, нажмите «📝 Оформить заявку»."

var dialog = fakeHistory{
	{Role: history.UserRole, Text: "Нужно утеплить балкон"},
	{Role: history.AIRole, Text: "Какая площадь?"},
	{Role: history.UserRole, Text: "6 квадратов"},
	{Role: history.AIRole, Text: finalAnswer},
}

func TestSubmitBuildsRequestFromDialog(t *testing.T) {
	s, repo, toManager, _ := newTestService(dialog, nil)

	reply, err := s.Submit(context.Background(), client, " +79991234567 ")
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if !strings.Contains(reply, "№1") || !strings.Contains(reply, "+79991234567") || !strings.Contains(reply, testItems) {
		t.Fatalf("reply = %q", reply)
	}

	want := Request{
		ID: 1, Client: client, Phone: "+79991234567", Status: StatusNew, Items: testItems,
		Question: "• Нужно утеплить балкон\n• 6 квадратов",
		Answer:   finalAnswer,
	}
	got := repo.created[0]
	got.CreatedAt = time.Time{}
	if got != want {
		t.Fatalf("created = %+v\nwant      %+v", got, want)
	}
	if len(*toManager) != 1 || (*toManager)[0].ID != 1 {
		t.Fatalf("manager got %+v", *toManager)
	}
}

func TestSubmitTakesLatestOrder(t *testing.T) {
	changed := append(slices.Clone(dialog),
		history.Message{Role: history.UserRole, Text: "А можно минвату?"},
		history.Message{Role: history.AIRole, Text: "Можно.\n\nИтого к заказу — утепление балкона 6 м²:\n\n- Knauf, минвата 50 мм или аналог — 1 упаковка\nЕсли всё подходит, нажмите кнопку."},
		history.Message{Role: history.UserRole, Text: "Спасибо"},
		history.Message{Role: history.AIRole, Text: "Пожалуйста! Итого к заказу остаётся прежним."},
	)
	s, repo, _, _ := newTestService(changed, nil)

	if _, err := s.Submit(context.Background(), client, "+79991234567"); err != nil {
		t.Fatal(err)
	}
	// The changed list wins; a later mention without a list doesn't count,
	// and the text right after the list isn't part of it.
	want := "Итого к заказу — утепление балкона 6 м²:\n- Knauf, минвата 50 мм или аналог — 1 упаковка"
	if got := repo.created[0].Items; got != want {
		t.Fatalf("Items = %q, want %q", got, want)
	}
}

func TestSubmitWithoutOrder(t *testing.T) {
	s, repo, toManager, _ := newTestService(dialog[:2], nil)

	reply, err := s.Submit(context.Background(), client, "+79991234567")
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	// The manager still gets the request and reads the dialog.
	if len(repo.created) != 1 || repo.created[0].Items != "" || len(*toManager) != 1 {
		t.Fatalf("created = %+v, manager got %d", repo.created, len(*toManager))
	}
	if !strings.Contains(reply, "принята") || strings.Contains(reply, "Итого") {
		t.Fatalf("reply = %q", reply)
	}
}

func TestSubmitWithoutDialog(t *testing.T) {
	s, repo, toManager, _ := newTestService(nil, nil)

	reply, err := s.Submit(context.Background(), client, "+79991234567")
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if len(repo.created) != 0 || len(*toManager) != 0 {
		t.Fatal("request created without a dialog")
	}
	if !strings.Contains(reply, "Сначала опишите") {
		t.Fatalf("reply = %q", reply)
	}
}

func TestSubmitKeepsRequestWhenManagerIsUnreachable(t *testing.T) {
	s, repo, _, _ := newTestService(dialog, errors.New("max is down"))

	reply, err := s.Submit(context.Background(), client, "+79991234567")
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if len(repo.created) != 1 || !strings.Contains(reply, "принята") {
		t.Fatalf("created = %d, reply = %q", len(repo.created), reply)
	}
}

func TestSetStatusNotifiesClientOnce(t *testing.T) {
	s, _, _, toClient := newTestService(dialog, nil)
	if _, err := s.Submit(context.Background(), client, "+79991234567"); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		if _, _, err := s.SetStatus(context.Background(), 1, StatusInProgress); err != nil {
			t.Fatalf("SetStatus() error = %v", err)
		}
	}

	if len(*toClient) != 1 || (*toClient)[0].to != client || !strings.Contains((*toClient)[0].text, "в работе") {
		t.Fatalf("client got %+v, want one «в работе» message", *toClient)
	}
}

func TestSetStatusFollowsTransitions(t *testing.T) {
	s, _, _, toClient := newTestService(dialog, nil)
	if _, err := s.Submit(context.Background(), client, "+79991234567"); err != nil {
		t.Fatal(err)
	}

	steps := []struct {
		status Status
		ok     bool
	}{
		{StatusIssued, false}, // can't be issued before it is ready
		{StatusReady, true},
		{StatusInProgress, false}, // no way back
		{StatusIssued, true},
		{StatusCancelled, false}, // closed
	}
	for _, step := range steps {
		_, ok, err := s.SetStatus(context.Background(), 1, step.status)
		if err != nil || ok != step.ok {
			t.Fatalf("SetStatus(%s) = %v, %v, want %v", step.status, ok, err, step.ok)
		}
	}

	if len(*toClient) != 2 || !strings.Contains((*toClient)[0].text, "можно забирать") || !strings.Contains((*toClient)[1].text, "выдана") {
		t.Fatalf("client got %+v", *toClient)
	}
}

func TestListShowsStatusAndPreview(t *testing.T) {
	s, _, _, _ := newTestService(dialog, nil)
	if _, err := s.Submit(context.Background(), client, "+79991234567"); err != nil {
		t.Fatal(err)
	}

	list, err := s.List(context.Background(), client)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	// 22:30 UTC is already the next day in Moscow.
	want := "№1 от 26.09.2026 — 🆕 Принята\nутепление балкона 6 м²"
	if !strings.Contains(list, want) {
		t.Fatalf("list = %q, want it to contain %q", list, want)
	}
}

func TestManagerTextShowsAnswerOnlyWithoutItems(t *testing.T) {
	req := Request{ID: 1, Phone: "+79991234567", Question: "• балкон", Answer: "Возьмите пеноплекс.", Items: testItems}
	// The items are a quote of the answer, so the answer itself is left out.
	if text := req.ManagerText(); !strings.Contains(text, testItems) || strings.Contains(text, "Консультант ответил") {
		t.Fatalf("with items: %q", text)
	}

	req.Items = ""
	if text := req.ManagerText(); !strings.Contains(text, "Консультант ответил:\nВозьмите пеноплекс.") {
		t.Fatalf("without items: %q", text)
	}
}
