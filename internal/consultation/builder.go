package consultation

import (
	"kubometr/internal/history"
	"strings"
)

func buildPrompt(messages []history.Message) string {
	var builder strings.Builder

	builder.WriteString(consultantPrompt)
	builder.WriteString("\n\nИстория диалога:\n\n")

	if len(messages) == 1 {
		builder.WriteString("Это первое сообщение пользователя.\n")
	} else {
		builder.WriteString("Диалог уже продолжается.\n")
		builder.WriteString("Не здоровайся повторно.\n")
	}

	builder.WriteString("\n")
	builder.WriteString("Ниже приведена полная история текущего диалога.\n")
	builder.WriteString("Используй ее как единственный источник контекста.\n")
	builder.WriteString("Не задавай повторно вопросы, ответы на которые уже есть в истории.\n")
	builder.WriteString("Если информации достаточно для рекомендации — сразу дай рекомендацию.\n")
	builder.WriteString("\n===== ИСТОРИЯ ДИАЛОГА =====\n\n")

	for _, msg := range messages {
		switch msg.Role {
		case history.UserRole:
			builder.WriteString("Пользователь:\n")
		case history.AIRole:
			builder.WriteString("Консультант:\n")
		}

		builder.WriteString(msg.Text)
		builder.WriteString("\n\n")
	}

	builder.WriteString("\n===== КОНЕЦ ИСТОРИИ =====\n")
	builder.WriteString("Ответь только на последнее сообщение пользователя.")

	return builder.String()
}
