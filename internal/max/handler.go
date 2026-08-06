package max

import (
	"encoding/json"
	"net/http"
)

type Handler struct {
	consultation ConsultationProcessor
}

func NewHandler(consultation ConsultationProcessor) *Handler {
	return &Handler{
		consultation: consultation,
	}
}

func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	const maxBytes = 1024 * 1024
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	var update Update
	err := json.NewDecoder(r.Body).Decode(&update)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request"))
		return
	}

	if update.Message == nil || update.Message.Body == nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK Request"))
		return
	}

	question := update.Message.Body.Text
	if question == "" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK Request"))
		return
	}

	ctx := r.Context()
	var chatID int64 = update.Message.Recipient.UserID

	answer, err := h.consultation.Process(ctx, chatID, question)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(answer))

}
