package max

const (
	updateMessageCreated = "message_created"
	updateBotStarted     = "bot_started"
)

type User struct {
	UserID int64  `json:"user_id"`
	Name   string `json:"name"`
	IsBot  bool   `json:"is_bot"`
}

type Message struct {
	Sender    *User        `json:"sender"`
	Recipient *Recipient   `json:"recipient"`
	Timestamp int64        `json:"timestamp"`
	Body      *MessageBody `json:"body"`
}

type MessageBody struct {
	Text string `json:"text"`
}

type Recipient struct {
	UserID   int64  `json:"user_id"`
	ChatID   int64  `json:"chat_id"`
	ChatType string `json:"chat_type"`
}

type Update struct {
	UpdateType string   `json:"update_type"`
	Timestamp  int64    `json:"timestamp"`
	Message    *Message `json:"message"`

	// Set for bot_started updates.
	ChatID int64 `json:"chat_id"`
	User   *User `json:"user"`
}
