package state

import (
	"sync"
)

type UserState string

const (
	StateIdle         UserState = "idle"
	StateConsultation UserState = "consultation"
)

type StateManager struct {
	mu     sync.RWMutex
	states map[int64]UserState
}

func New() *StateManager {
	return &StateManager{
		states: make(map[int64]UserState),
	}
}

func (sm *StateManager) Set(chatID int64, state UserState) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.states[chatID] = state
}

func (sm *StateManager) Get(chatID int64) UserState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	state, ok := sm.states[chatID]
	if !ok {
		return StateIdle
	}
	return state
}

func (sm *StateManager) Delete(chatID int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.states, chatID)
}
