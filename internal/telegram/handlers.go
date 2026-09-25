package telegram

import (
	"context"
	"log/slog"

	"kubometr/internal/chat"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const telegramMessageLimit = 4096

const (
	consultationText = `Здравствуйте!

Я — виртуальный консультант Кубометра.

Опишите, что вы хотите сделать, а я помогу подобрать материалы.

Например:

• Нужно утеплить балкон.
• Хочу сделать перегородку из гипсокартона.
• Нужна краска для ванной.
• Планирую залить стяжку пола.`

	helpText = `📖 Как пользоваться ботом

` + chat.ButtonConsultation + ` — опишите задачу, например «нужно утеплить балкон 6 м²», и я подскажу, какие материалы понадобятся.
` + chat.ButtonSubmit + ` — менеджер перезвонит и уточнит цены, наличие и доставку. В заявку попадёт ваш диалог с консультантом, а номер телефона Telegram попросит подтвердить.
` + chat.ButtonRequests + ` — статусы ваших заявок.
` + chat.ButtonManager + ` — как связаться с менеджером.
` + chat.ButtonNewDialog + ` — начать заново, история очищается.

Команды: /start, /new, /requests, /manager, /help`

	errorText = "Что-то пошло не так. Попробуйте ещё раз чуть позже."
)

// mainKeyboard stays under the input field. The request button asks Telegram
// for the client's phone number, which arrives as a contact message.
var mainKeyboard = &models.ReplyKeyboardMarkup{
	ResizeKeyboard: true,
	Keyboard: [][]models.KeyboardButton{
		{{Text: chat.ButtonConsultation}, {Text: chat.ButtonSubmit, RequestContact: true}},
		{{Text: chat.ButtonRequests}, {Text: chat.ButtonManager}},
		{{Text: chat.ButtonNewDialog}, {Text: chat.ButtonHelp}},
	},
}

var commands = []models.BotCommand{
	{Command: "start", Description: "Главное меню"},
	{Command: "new", Description: "Новый диалог"},
	{Command: "requests", Description: "Мои заявки"},
	{Command: "manager", Description: "Связаться с менеджером"},
	{Command: "help", Description: "Помощь"},
}

func (t *Telegram) HandleStart(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	if err := t.consultation.Reset(ctx, chatRef(chatID)); err != nil {
		slog.ErrorContext(ctx, "reset consultation", "chat_id", chatID, "error", err)
	}

	t.sendWithKeyboard(ctx, chatID, "Добро пожаловать в Кубометр!\n\nВыберите действие:")
}

func (t *Telegram) HandleHelp(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	// The keyboard is sent again, so chats started before new buttons
	// appeared get them too.
	t.sendWithKeyboard(ctx, update.Message.Chat.ID, helpText)
}

func (t *Telegram) HandleConsultation(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	t.consultation.Start(chatRef(chatID))
	t.reply(ctx, chatID, consultationText)
}

func (t *Telegram) HandleNewDialog(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	if err := t.consultation.Reset(ctx, chatRef(chatID)); err != nil {
		slog.ErrorContext(ctx, "reset consultation", "chat_id", chatID, "error", err)
	}
	t.consultation.Start(chatRef(chatID))
	t.sendWithKeyboard(ctx, chatID, consultationText)
}

func (t *Telegram) HandleRequests(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	list, err := t.requests.List(ctx, chatRef(chatID))
	if err != nil {
		slog.ErrorContext(ctx, "list requests", "chat_id", chatID, "error", err)
		list = errorText
	}
	t.reply(ctx, chatID, list)
}

func (t *Telegram) HandleManager(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	t.reply(ctx, update.Message.Chat.ID, chat.ManagerText(t.managerContact))
}

func hasContact(update *models.Update) bool {
	return update.Message != nil && update.Message.Contact != nil
}

// HandleContact turns a shared phone number into a request.
func (t *Telegram) HandleContact(ctx context.Context, b *bot.Bot, update *models.Update) {
	chatID := update.Message.Chat.ID
	answer, err := t.requests.Submit(ctx, chatRef(chatID), update.Message.Contact.PhoneNumber)
	if err != nil {
		slog.ErrorContext(ctx, "submit request", "chat_id", chatID, "error", err)
		answer = errorText
	}
	t.reply(ctx, chatID, answer)
}

func (t *Telegram) HandleMessage(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	question := update.Message.Text

	stopTyping := chat.KeepTyping(ctx, chat.TypingInterval, func(ctx context.Context) error {
		_, err := b.SendChatAction(ctx, &bot.SendChatActionParams{
			ChatID: chatID,
			Action: models.ChatActionTyping,
		})
		return err
	})
	answer, err := t.consultation.Process(ctx, chatRef(chatID), question)
	stopTyping()

	if err != nil {
		slog.ErrorContext(ctx, "process consultation", "chat_id", chatID, "error", err)
		answer = "Не удалось получить ответ от AI-консультанта. Попробуйте повторить вопрос чуть позже."
	}
	t.reply(ctx, chatID, answer)
}

// SendText sends a text of any length, split into several messages if needed.
func (t *Telegram) SendText(ctx context.Context, chatID int64, text string) error {
	for _, part := range chat.SplitText(text, telegramMessageLimit) {
		_, err := t.bot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   part,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// reply sends a text and logs a failure: the handler has nobody to return
// the error to.
func (t *Telegram) reply(ctx context.Context, chatID int64, text string) {
	if err := t.SendText(ctx, chatID, text); err != nil {
		slog.ErrorContext(ctx, "send message", "chat_id", chatID, "error", err)
	}
}

func (t *Telegram) sendWithKeyboard(ctx context.Context, chatID int64, text string) {
	_, err := t.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: mainKeyboard,
	})
	if err != nil {
		slog.ErrorContext(ctx, "send message", "chat_id", chatID, "error", err)
	}
}

func chatRef(chatID int64) chat.ID {
	return chat.ID{Platform: chat.Telegram, ChatID: chatID}
}
