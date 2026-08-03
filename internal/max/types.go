package max

type Message struct {
	Sender    interface{}  `json:"sender"`
	Recipient *Recipient   `json:"recipient"`
	Timestamp int64        `json:"timestamp"`
	Body      *MessageBody `json:"body"`
}

type MessageBody struct {
	Text string `json:"text"`
}

type Recipient struct {
	UserID int64  `json:"user_id"`
	ChatID string `json:"chat_id"`
}

type Update struct {
	UpdateType string   `json:"update_type"`
	Timestamp  int64    `json:"timestamp"`
	Message    *Message `json:"message"`
}
