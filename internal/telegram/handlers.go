package telegram

import (
	"context"
	"log/slog"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	buttonConsultation = "💬 Консультация"
	buttonRequests     = "🧾 Мои заявки"
	buttonHelp         = "ℹ️ Помощь"

	telegramMessageLimit = 4096
)

func (t *Telegram) HandleStart(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	t.consultation.Reset(chatID)

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   "Добро пожаловать в Кубометр!\n\nВыберите действие:",

		ReplyMarkup: &models.ReplyKeyboardMarkup{
			ResizeKeyboard: true,
			Keyboard: [][]models.KeyboardButton{
				{
					{
						Text: buttonConsultation,
					},
					{
						Text: buttonRequests,
					},
				},
				{
					{
						Text: buttonHelp,
					},
				},
			},
		},
	})

	if err != nil {
		slog.ErrorContext(ctx, "send start message", "chat_id", chatID, "error", err)
	}
}

func (t *Telegram) HandleHelp(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text: `📖 Доступные команды:

	/start - начать работу
	/help - показать помощь`,
	})

	if err != nil {
		slog.ErrorContext(ctx, "send help message", "chat_id", update.Message.Chat.ID, "error", err)
	}
}

func (t *Telegram) HandleConsultation(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	t.consultation.Start(chatID)

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text: `Здравствуйте!

Я — виртуальный консультант Кубометра.

Опишите, что вы хотите сделать, а я помогу подобрать материалы.

Например:

• Нужно утеплить балкон.
• Хочу сделать перегородку из гипсокартона.
• Нужна краска для ванной.
• Планирую залить стяжку пола.`,
	})
	if err != nil {
		slog.ErrorContext(ctx, "send consultation message", "chat_id", chatID, "error", err)
	}
}

func (t *Telegram) HandleRequests(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	err := t.sendText(ctx, b, chatID, "Раздел заявок пока в разработке. Если нужна помощь с подбором материалов, нажмите «💬 Консультация».")
	if err != nil {
		slog.ErrorContext(ctx, "send requests message", "chat_id", chatID, "error", err)
	}
}

func (t *Telegram) HandleMessage(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	question := update.Message.Text

	answer, err := t.consultation.Process(ctx, chatID, question)
	if err != nil {
		slog.ErrorContext(ctx, "process consultation", "chat_id", chatID, "error", err)

		sendErr := t.sendText(
			ctx,
			b,
			chatID,
			"Не удалось получить ответ от AI-консультанта. Попробуйте повторить вопрос чуть позже.",
		)
		if sendErr != nil {
			slog.ErrorContext(ctx, "send fallback message", "chat_id", chatID, "error", sendErr)
		}

		return
	}

	if err := t.sendText(ctx, b, chatID, answer); err != nil {
		slog.ErrorContext(ctx, "send answer", "chat_id", chatID, "error", err)
	}
}

func (t *Telegram) sendText(ctx context.Context, b *bot.Bot, chatID int64, text string) error {
	for _, part := range splitMessage(text, telegramMessageLimit) {
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   part,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func splitMessage(text string, limit int) []string {
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