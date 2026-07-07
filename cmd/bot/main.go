package main

import (
	"log/slog"
	"os"

	"kubometr/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		slog.Error("application failed", "error", err)
		os.Exit(1)
	}
}
