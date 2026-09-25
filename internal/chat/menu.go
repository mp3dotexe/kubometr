package chat

// Menu buttons. Pressing one sends its label as a message, so both
// messengers match incoming text against these.
const (
	ButtonConsultation = "💬 Консультация"
	ButtonSubmit       = "📝 Оформить заявку"
	ButtonRequests     = "🧾 Мои заявки"
	ButtonManager      = "📞 Менеджер"
	ButtonNewDialog    = "🔄 Новый диалог"
	ButtonHelp         = "ℹ️ Помощь"
)

// ManagerText tells the client how to reach a manager.
func ManagerText(contact string) string {
	text := "Менеджер ответит на вопросы о ценах, наличии и доставке."
	if contact != "" {
		text += "\n\n📞 " + contact
	}
	return text + "\n\nМожно и не звонить: нажмите «" + ButtonSubmit + "», и менеджер перезвонит сам."
}
