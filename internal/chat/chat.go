package chat

// Platform is the messenger a chat belongs to.
type Platform string

const (
	Telegram Platform = "telegram"
	MAX      Platform = "max"
)

// ID identifies a chat across messengers. Chat IDs of different messengers
// may collide, so the platform is part of the key.
type ID struct {
	Platform Platform
	ChatID   int64
}

// SplitText splits text into parts of at most limit runes each.
func SplitText(text string, limit int) []string {
	if text == "" {
		return []string{""}
	}
	if limit <= 0 {
		return []string{text}
	}

	runes := []rune(text)
	parts := make([]string, 0, (len(runes)/limit)+1)
	for len(runes) > limit {
		parts = append(parts, string(runes[:limit]))
		runes = runes[limit:]
	}
	parts = append(parts, string(runes))

	return parts
}
