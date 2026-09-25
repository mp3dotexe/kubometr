package consultation

import (
	"kubometr/internal/ai"
	"kubometr/internal/history"
)

const continuationNote = "Диалог уже продолжается — не здоровайся повторно. " +
	"Не задавай вопросы, ответы на которые уже есть в предыдущих сообщениях."

// buildMessages turns the stored dialog into a chat completion request:
// the consultant instructions go into the system message and every stored
// message keeps its role, so the model sees a real multi-turn conversation.
func buildMessages(messages []history.Message) []ai.Message {
	system := consultantPrompt
	if len(messages) > 1 {
		system += "\n\n" + continuationNote
	}

	result := make([]ai.Message, 0, len(messages)+1)
	result = append(result, ai.Message{Role: ai.RoleSystem, Content: system})

	for _, msg := range messages {
		switch msg.Role {
		case history.UserRole:
			result = append(result, ai.Message{Role: ai.RoleUser, Content: msg.Text})
		case history.AIRole:
			result = append(result, ai.Message{Role: ai.RoleAssistant, Content: msg.Text})
		}
	}

	return result
}
