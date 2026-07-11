package history

import (
	"sync"
	"time"
)

type Role string

const (
	UserRole Role = "user"
	AIRole   Role = "ai"
)

type Message struct {
	Role Role
	Text string
	Time time.Time
}

type HistoryManager struct {
	mu        sync.RWMutex
	histories map[int64][]Message
}

func New() *HistoryManager {
	return &HistoryManager{
		histories: make(map[int64][]Message),
	}
}

func (hm *HistoryManager) Add(chatID int64, msg Message) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.histories[chatID] = append(hm.histories[chatID], msg)

	if len(hm.histories[chatID]) > 20 {
		hm.histories[chatID] = hm.histories[chatID][1:]
	}
}

func (hm *HistoryManager) Get(chatID int64) []Message {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	history := hm.histories[chatID]

	copyHistory := make([]Message, len(history))
	copy(copyHistory, history)

	return copyHistory
}

func (hm *HistoryManager) Delete(chatID int64) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	delete(hm.histories, chatID)
}
