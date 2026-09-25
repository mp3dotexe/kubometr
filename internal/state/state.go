package state

import (
	"sync"

	"kubometr/internal/chat"
)

type UserState string

const (
	StateIdle         UserState = "idle"
	StateConsultation UserState = "consultation"
)

type StateManager struct {
	mu     sync.RWMutex
	states map[chat.ID]UserState
}

func New() *StateManager {
	return &StateManager{
		states: make(map[chat.ID]UserState),
	}
}

func (sm *StateManager) Set(id chat.ID, state UserState) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.states[id] = state
}

func (sm *StateManager) Get(id chat.ID) UserState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	state, ok := sm.states[id]
	if !ok {
		return StateIdle
	}
	return state
}

func (sm *StateManager) Delete(id chat.ID) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.states, id)
}
