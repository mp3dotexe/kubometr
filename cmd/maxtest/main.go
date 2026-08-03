package maxtest

import (
	"kubometr/internal/max"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", max.NewHandler(nil).HandleWebhook)
	http.ListenAndServe(":8080", nil)
}
