package max

import (
	"encoding/json"
	"strings"
)

const (
	updateMessageCreated  = "message_created"
	updateBotStarted      = "bot_started"
	updateMessageCallback = "message_callback"
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
	Mid         string       `json:"mid"`
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments"`
}

type Attachment struct {
	Type string `json:"type"`
	// Payload differs by attachment type, so it is decoded only for contacts.
	Payload json.RawMessage `json:"payload"`
}

// ContactPhone returns the phone number of a shared contact, or "" when the
// message has none. MAX sends the contact as a vCard.
func (b *MessageBody) ContactPhone() string {
	for _, a := range b.Attachments {
		if a.Type != "contact" {
			continue
		}
		var payload struct {
			VcfInfo string `json:"vcf_info"`
		}
		if json.Unmarshal(a.Payload, &payload) != nil {
			continue
		}
		for _, line := range strings.Split(payload.VcfInfo, "\n") {
			// TEL;TYPE=cell:+79991234567 or TEL;VALUE=uri:tel:+79991234567
			if strings.HasPrefix(strings.ToUpper(line), "TEL") {
				return strings.TrimSpace(line[strings.LastIndex(line, ":")+1:])
			}
		}
	}
	return ""
}

// Callback is a pressed inline button.
type Callback struct {
	CallbackID string `json:"callback_id"`
	Payload    string `json:"payload"`
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

	// Set for message_callback updates; Message is then the bot's message
	// with the pressed button.
	Callback *Callback `json:"callback"`

	// Set for bot_started updates.
	ChatID int64 `json:"chat_id"`
	User   *User `json:"user"`
}
