package history

import (
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