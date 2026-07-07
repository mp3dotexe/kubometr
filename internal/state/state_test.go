package state

import "testing"

func TestStateManagerDefaultsToIdle(t *testing.T) {
	sm := New()

	if got := sm.Get(42); got != StateIdle {
		t.Fatalf("Get() = %q, want %q", got, StateIdle)
	}
}

func TestStateManagerSetAndDelete(t *testing.T) {
	sm := New()

	sm.Set(42, StateConsultation)
	if got := sm.Get(42); got != StateConsultation {
		t.Fatalf("Get() after Set() = %q, want %q", got, StateConsultation)
	}

	sm.Delete(42)
	if got := sm.Get(42); got != StateIdle {
		t.Fatalf("Get() after Delete() = %q, want %q", got, StateIdle)
	}
}
