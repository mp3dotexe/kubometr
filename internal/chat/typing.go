package chat

import (
	"context"
	"time"
)

// TypingInterval is how often the typing status is refreshed: messengers
// show it for about five seconds after each request.
const TypingInterval = 4 * time.Second

// KeepTyping calls sendTyping right away and then every interval until the
// returned stop function is called. Errors are ignored: the typing status is
// cosmetic and must never block the answer.
func KeepTyping(ctx context.Context, interval time.Duration, sendTyping func(context.Context) error) (stop func()) {
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	go func() {
		defer close(done)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			_ = sendTyping(ctx)

			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()

	return func() {
		cancel()
		<-done
	}
}
