package chat

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestKeepTypingRepeatsUntilStopped(t *testing.T) {
	var calls atomic.Int32
	stop := KeepTyping(context.Background(), 10*time.Millisecond, func(context.Context) error {
		calls.Add(1)
		return errors.New("ignored")
	})

	time.Sleep(55 * time.Millisecond)
	stop()
	afterStop := calls.Load()

	if afterStop < 3 {
		t.Fatalf("typing sent %d times, want at least 3", afterStop)
	}

	time.Sleep(30 * time.Millisecond)
	if calls.Load() != afterStop {
		t.Fatal("typing still sent after stop")
	}
}

func TestKeepTypingSendsImmediately(t *testing.T) {
	sent := make(chan struct{}, 1)
	stop := KeepTyping(context.Background(), time.Hour, func(context.Context) error {
		sent <- struct{}{}
		return nil
	})
	defer stop()

	select {
	case <-sent:
	case <-time.After(time.Second):
		t.Fatal("typing was not sent right away")
	}
}
