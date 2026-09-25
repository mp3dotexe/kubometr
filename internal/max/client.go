package max

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"kubometr/internal/chat"
	"kubometr/internal/requests"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

// Menu is the inline keyboard attached to a message. MAX has no persistent
// reply keyboard, so the menu goes under every message to a client, the way
// Telegram keeps it under the input field; a message button sends its label
// as the client's message, like a Telegram button.
type Menu int

const (
	NoMenu Menu = iota
	MainMenu
)

func (m Menu) keyboard() *maxbot.Keyboard {
	if m != MainMenu {
		return nil
	}
	return maxbot.InlineKeyboard(
		maxbot.Row(maxbot.BtnContact(chat.ButtonSubmit)),
		maxbot.Row(maxbot.BtnMsg(chat.ButtonRequests), maxbot.BtnMsg(chat.ButtonManager)),
		maxbot.Row(maxbot.BtnMsg(chat.ButtonNewDialog), maxbot.BtnMsg(chat.ButtonHelp)),
	)
}

type Client struct {
	api *maxbot.Api
}

func NewClient(token string, httpClient *http.Client) (*Client, error) {
	mx, err := maxbot.New(token, maxbot.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create max client: %w", err)
	}
	return &Client{api: mx}, nil
}

func (c *Client) SendMessage(ctx context.Context, chatID int64, text string, menu Menu) error {
	msg := maxbot.NewMessage().SetChat(chatID).SetText(text)
	if kb := menu.keyboard(); kb != nil {
		msg.AddKeyboard(kb)
	}
	err := c.api.Messages.Send(ctx, msg)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	return nil
}

func (c *Client) SendTyping(ctx context.Context, chatID int64) error {
	result, err := c.api.Chats.SendAction(ctx, chatID, schemes.TYPING_ON)
	if err != nil {
		return fmt.Errorf("send typing: %w", err)
	}
	if !result.Success {
		return errors.New("send typing: " + result.Message)
	}
	return nil
}

// SendRequest delivers a new request to the manager chat, with buttons to
// change its status.
func (c *Client) SendRequest(ctx context.Context, chatID int64, req requests.Request) error {
	body := requestMessage(req)
	msg := maxbot.NewMessage().SetChat(chatID).SetText(body.Text)
	if kb := statusKeyboard(req); kb != nil {
		msg.AddKeyboard(kb)
	}
	if err := c.api.Messages.Send(ctx, msg); err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	return nil
}

// AnswerCallback confirms a button press with a short notification. When
// req is not nil, the manager's message is redrawn with its new status.
func (c *Client) AnswerCallback(ctx context.Context, callbackID string, req *requests.Request, notification string) error {
	answer := &schemes.CallbackAnswer{Notification: notification}
	if req != nil {
		answer.Message = requestMessage(*req)
	}
	result, err := c.api.Messages.AnswerOnCallback(ctx, callbackID, answer)
	if err != nil {
		return fmt.Errorf("answer callback: %w", err)
	}
	if !result.Success {
		return errors.New("answer callback: " + result.Message)
	}
	return nil
}

func requestMessage(req requests.Request) *schemes.NewMessageBody {
	// The status buttons must stay attached, so the text can't be split.
	text := req.ManagerText()
	if runes := []rune(text); len(runes) > maxMessageLimit {
		text = string(runes[:maxMessageLimit-1]) + "…"
	}
	body := &schemes.NewMessageBody{Text: text, Attachments: []any{}}
	if kb := statusKeyboard(req); kb != nil {
		body.Attachments = append(body.Attachments, schemes.NewInlineKeyboardAttachmentRequest(kb.Build()))
	}
	return body
}

// statusKeyboard offers the statuses a request can still move to, with
// cancelling on its own row so it isn't pressed by accident.
func statusKeyboard(req requests.Request) *maxbot.Keyboard {
	var row, cancelRow []schemes.ButtonInterface
	for _, status := range req.Status.Next() {
		payload := statusPayload(req.ID, status)
		switch status {
		case requests.StatusCancelled:
			cancelRow = append(cancelRow, maxbot.Btn(status.Title(), payload, schemes.NEGATIVE))
		case requests.StatusInProgress:
			row = append(row, maxbot.Btn(status.Title(), payload))
		default:
			row = append(row, maxbot.Btn(status.Title(), payload, schemes.POSITIVE))
		}
	}
	if row == nil && cancelRow == nil {
		return nil
	}
	return maxbot.InlineKeyboard(row, cancelRow)
}

const statusPayloadPrefix = "request:"

func statusPayload(id int64, status requests.Status) string {
	return statusPayloadPrefix + strconv.FormatInt(id, 10) + ":" + string(status)
}

// parseStatusPayload is the reverse of statusPayload.
func parseStatusPayload(payload string) (id int64, status requests.Status, ok bool) {
	rest, found := strings.CutPrefix(payload, statusPayloadPrefix)
	if !found {
		return 0, "", false
	}
	idText, statusText, found := strings.Cut(rest, ":")
	id, err := strconv.ParseInt(idText, 10, 64)
	if !found || err != nil {
		return 0, "", false
	}
	// Whether the request may move to this status is checked when it's saved.
	return id, requests.Status(statusText), true
}

// Subscribe registers webhookURL so MAX starts delivering updates to it.
// Registering the same URL again just updates the subscription.
func (c *Client) Subscribe(ctx context.Context, webhookURL, secret string) error {
	result, err := c.api.Subscriptions.Subscribe(
		ctx,
		webhookURL,
		[]string{updateMessageCreated, updateBotStarted, updateMessageCallback},
		secret,
	)
	if err != nil {
		return fmt.Errorf("subscribe webhook: %w", err)
	}
	if !result.Success {
		return errors.New("subscribe webhook: " + result.Message)
	}
	return nil
}
