package max

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"errors"
	"context"
)

type mockConsultation struct {
	answer string
	err error
}

func TestHandleWebhook_InvalidJSON(t *testing.T) {

	handler := NewHandler(nil)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("invalid json"))
	recorder := httptest.NewRecorder()

	handler.HandleWebhook(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("expected status code %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

func (m *mockConsultation) Process(ctx context.Context, chatID int64, question string) (string, error) {
	return m.answer, m.err
}

func TestHandleWebhook_ConsultationError(t *testing.T) {
	handler := NewHandler(&mockConsultation{
		answer: "",
		err: errors.New("consultation error"),
	})

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{"update_type": "message_created", "message": {"body": {"text": "test question"}, "recipient": {"user_id": 12345}}}`))
	recorder := httptest.NewRecorder()

	handler.HandleWebhook(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("expected status code %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
}

func TestHandleWebhook_Success(t *testing.T) {
	handler := NewHandler(&mockConsultation{
		answer: "test answer",
		err: nil,
	})

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(`{"update_type": "message_created", "message": {"body": {"text": "test question"}, "recipient": {"user_id": 12345}}}`))
	recorder := httptest.NewRecorder()

	handler.HandleWebhook(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, recorder.Code)
	}
	recordedBody := recorder.Body.String()
	if !strings.Contains(recordedBody, "test answer") {
		t.Errorf("expected response body to contain %q, got %q", "test answer", recordedBody)
	}
}
