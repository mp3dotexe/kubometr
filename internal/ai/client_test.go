package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeAPI emulates a chat completions endpoint: each model either answers
// with its text or fails with its HTTP status.
type fakeAPI struct {
	delays   map[string]time.Duration
	mu       sync.Mutex
	answers  map[string]string
	statuses map[string]int
	calls    []string
	received []map[string]any
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Model    string           `json:"model"`
		Messages []map[string]any `json:"messages"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	f.mu.Lock()
	f.calls = append(f.calls, req.Model)
	f.received = req.Messages
	status, failing := f.statuses[req.Model]
	answer := f.answers[req.Model]
	delay := f.delays[req.Model]
	f.mu.Unlock()

	select {
	case <-time.After(delay):
	case <-r.Context().Done():
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if failing {
		w.WriteHeader(status)
		fmt.Fprintf(w, `{"error": {"message": "fail %d", "code": %d}}`, status, status)
		return
	}

	content, _ := json.Marshal(answer)
	fmt.Fprintf(w, `{"id": "1", "object": "chat.completion", "model": %q, "choices": [
		{"index": 0, "finish_reason": "stop", "message": {"role": "assistant", "content": %s}}]}`,
		req.Model, content)
}

func newTestClient(t *testing.T, api *fakeAPI, modelTimeout time.Duration, models ...string) *Client {
	t.Helper()
	srv := httptest.NewServer(api)
	t.Cleanup(srv.Close)

	c, err := New("test-key", srv.URL, models, modelTimeout)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return c
}

var question = []Message{
	{Role: RoleSystem, Content: "ты консультант"},
	{Role: RoleUser, Content: "нужна краска"},
}

func TestNewRequiresModel(t *testing.T) {
	if _, err := New("key", "http://localhost", nil, time.Second); err == nil {
		t.Fatal("New() error = nil, want error without models")
	}
}

func TestCompleteSendsRolesToFirstModel(t *testing.T) {
	api := &fakeAPI{answers: map[string]string{"a": "ответ"}}
	c := newTestClient(t, api, time.Second, "a", "b")

	answer, err := c.Complete(context.Background(), question)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if answer != "ответ" {
		t.Fatalf("answer = %q", answer)
	}
	if len(api.calls) != 1 || api.calls[0] != "a" {
		t.Fatalf("calls = %v, want [a]", api.calls)
	}
	if len(api.received) != 2 || api.received[0]["role"] != "system" || api.received[1]["role"] != "user" {
		t.Fatalf("received messages = %v", api.received)
	}
}

func TestCompleteFallsBack(t *testing.T) {
	cases := map[string]*fakeAPI{
		"rate limited":  {statuses: map[string]int{"a": http.StatusTooManyRequests}},
		"model removed": {statuses: map[string]int{"a": http.StatusNotFound}},
		"server error":  {statuses: map[string]int{"a": http.StatusBadGateway}},
		"empty answer":  {answers: map[string]string{"a": "  "}},
	}

	for name, api := range cases {
		t.Run(name, func(t *testing.T) {
			if api.answers == nil {
				api.answers = map[string]string{}
			}
			api.answers["b"] = "ответ b"
			c := newTestClient(t, api, time.Second, "a", "b")

			answer, err := c.Complete(context.Background(), question)
			if err != nil {
				t.Fatalf("Complete() error = %v", err)
			}
			if answer != "ответ b" {
				t.Fatalf("answer = %q, want answer from fallback model", answer)
			}
			// The failing model must not be retried before falling back.
			if strings.Join(api.calls, ",") != "a,b" {
				t.Fatalf("calls = %v, want [a b]", api.calls)
			}
		})
	}
}

func TestCompleteAllModelsFail(t *testing.T) {
	api := &fakeAPI{statuses: map[string]int{"a": http.StatusTooManyRequests, "b": http.StatusNotFound}}
	c := newTestClient(t, api, time.Second, "a", "b")

	_, err := c.Complete(context.Background(), question)
	if err == nil {
		t.Fatal("Complete() error = nil, want error")
	}
	for _, model := range []string{"a:", "b:"} {
		if !strings.Contains(err.Error(), model) {
			t.Errorf("error %q does not mention model %q", err, model)
		}
	}
}

func TestCompleteStopsOnInvalidKey(t *testing.T) {
	api := &fakeAPI{
		statuses: map[string]int{"a": http.StatusUnauthorized},
		answers:  map[string]string{"b": "ответ b"},
	}
	c := newTestClient(t, api, time.Second, "a", "b")

	if _, err := c.Complete(context.Background(), question); err == nil {
		t.Fatal("Complete() error = nil, want error")
	}
	if len(api.calls) != 1 {
		t.Fatalf("calls = %v, want only the first model", api.calls)
	}
}

func TestCompleteStopsWhenContextDone(t *testing.T) {
	api := &fakeAPI{answers: map[string]string{"a": "ответ"}}
	c := newTestClient(t, api, time.Second, "a", "b")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := c.Complete(ctx, question); err == nil {
		t.Fatal("Complete() error = nil, want error")
	}
	if len(api.calls) != 0 {
		t.Fatalf("calls = %v, want none", api.calls)
	}
}

func TestCompleteRejectsUnknownRole(t *testing.T) {
	c := newTestClient(t, &fakeAPI{}, time.Second, "a")

	if _, err := c.Complete(context.Background(), []Message{{Role: "tool", Content: "x"}}); err == nil {
		t.Fatal("Complete() error = nil, want error")
	}
}

func TestCompleteFallsBackFromStuckModel(t *testing.T) {
	api := &fakeAPI{
		delays:  map[string]time.Duration{"a": 5 * time.Second},
		answers: map[string]string{"a": "поздно", "b": "ответ b"},
	}
	c := newTestClient(t, api, 100*time.Millisecond, "a", "b")

	start := time.Now()
	answer, err := c.Complete(context.Background(), question)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if answer != "ответ b" {
		t.Fatalf("answer = %q, want answer from fallback model", answer)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("took %v, the stuck model was not cut off", elapsed)
	}
}

func TestCompleteLastModelIgnoresModelTimeout(t *testing.T) {
	api := &fakeAPI{
		delays:  map[string]time.Duration{"a": 300 * time.Millisecond},
		answers: map[string]string{"a": "медленный ответ"},
	}
	c := newTestClient(t, api, 100*time.Millisecond, "a")

	answer, err := c.Complete(context.Background(), question)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if answer != "медленный ответ" {
		t.Fatalf("answer = %q", answer)
	}
}
