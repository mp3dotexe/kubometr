package state

import (
	"testing"

	"kubometr/internal/chat"
)

var testChat = chat.ID{Platform: chat.Telegram, ChatID: 42}

func TestStateManagerDefaultsToIdle(t *testing.T) {
	sm := New()

	if got := sm.Get(testChat); got != StateIdle {
		t.Fatalf("Get() = %q, want %q", got, StateIdle)
	}
}

func TestStateManagerSetAndDelete(t *testing.T) {
	sm := New()

	sm.Set(testChat, StateConsultation)
	if got := sm.Get(testChat); got != StateConsultation {
		t.Fatalf("Get() after Set() = %q, want %q", got, StateConsultation)
	}

	sm.Delete(testChat)
	if got := sm.Get(testChat); got != StateIdle {
		t.Fatalf("Get() after Delete() = %q, want %q", got, StateIdle)
	}
}

func TestStateManagerSeparatesPlatforms(t *testing.T) {
	sm := New()

	sm.Set(chat.ID{Platform: chat.Telegram, ChatID: 1}, StateConsultation)

	if got := sm.Get(chat.ID{Platform: chat.MAX, ChatID: 1}); got != StateIdle {
		t.Fatalf("MAX chat state = %q, want %q", got, StateIdle)
	}
}
