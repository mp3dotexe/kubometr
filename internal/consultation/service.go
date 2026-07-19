package consultation

import (
	"context"
	"errors"
	"fmt"
	"kubometr/internal/ai"
	"kubometr/internal/history"
	"kubometr/internal/state"
	"kubometr/internal/users"
	"strings"
	"sync"
	"time"
)

var ErrUnknownUserState = errors.New("unknown user state")

const historyLimit = 20

type Service struct {
	state           *state.StateManager
	ai              *ai.Client
	aiTimeout       time.Duration
	aiRateLimit     time.Duration
	maxPromptLength int
	aiLimiter       chan struct{}
	mu              sync.Mutex
	lastAIRequest   map[int64]time.Time
	history         *history.Repository
	users           *users.Repository
}

func New(
	state *state.StateManager,
	ai *ai.Client,
	aiTimeout time.Duration,
	aiRateLimit time.Duration,
	maxPromptLength int,
	maxConcurrentAI int,
	history *history.Repository,
	users *users.Repository,
) *Service {
	return &Service{
		state:           state,
		ai:              ai,
		aiTimeout:       aiTimeout,
		aiRateLimit:     aiRateLimit,
		maxPromptLength: maxPromptLength,
		aiLimiter:       make(chan struct{}, maxConcurrentAI),
		lastAIRequest:   make(map[int64]time.Time),
		history:         history,
		users:           users,
	}
}

func (s *Service) Process(ctx context.Context, chatID int64, question string) (string, error) {
	userState := s.state.Get(chatID)

	userID, err := s.users.GetOrCreate(ctx, chatID)
	if err != nil {
		return "", fmt.Errorf("get or create user: %w", err)
	}

	switch userState {
	case state.StateIdle:
		return "Сначала нажмите кнопку «💬 Консультация».", nil

	case state.StateConsultation:
		question = strings.TrimSpace(question)
		if question == "" {
			return "Опишите задачу текстом, и я помогу подобрать материалы.", nil
		}

		if len([]rune(question)) > s.maxPromptLength {
			return "Сообщение слишком длинное. Сформулируйте задачу короче и отправьте ее одним сообщением.", nil
		}

		if !s.canAskAI(chatID, time.Now()) {
			return "Пожалуйста, подождите несколько секунд перед следующим вопросом.", nil
		}

		select {
		case s.aiLimiter <- struct{}{}:
			defer func() { <-s.aiLimiter }()
		default:
			return "Сейчас много запросов. Попробуйте еще раз через минуту.", nil
		}

		if err := s.history.Save(ctx, userID, string(history.UserRole), question); err != nil {
			return "", fmt.Errorf("save user message: %w", err)
		}

		messages, err := s.history.LoadHistory(ctx, userID, historyLimit)
		if err != nil {
			return "", fmt.Errorf("load history: %w", err)
		}
		prompt := buildPrompt(messages)

		aiCtx, cancel := context.WithTimeout(ctx, s.aiTimeout)
		defer cancel()

		answer, err := s.ai.Ask(aiCtx, prompt)
		if err != nil {
			return "", err
		}

		answer = strings.TrimSpace(answer)
		if answer == "" {
			return "AI-консультант вернул пустой ответ. Попробуйте переформулировать вопрос.", nil
		}

		if err := s.history.Save(ctx, userID, string(history.AIRole), answer); err != nil {
			return "", fmt.Errorf("save ai response: %w", err)
		}

		return answer, nil
	}

	return "", ErrUnknownUserState
}

func (s *Service) canAskAI(chatID int64, now time.Time) bool {
	if s.aiRateLimit <= 0 {
		return true
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	lastRequestAt, ok := s.lastAIRequest[chatID]
	if ok && now.Sub(lastRequestAt) < s.aiRateLimit {
		return false
	}

	s.lastAIRequest[chatID] = now
	return true
}

func (s *Service) Start(chatID int64) {
	s.state.Set(chatID, state.StateConsultation)
}

func (s *Service) Reset(ctx context.Context, chatID int64) error {
	s.state.Delete(chatID)

	s.mu.Lock()
	delete(s.lastAIRequest, chatID)
	s.mu.Unlock()

	userID, err := s.users.GetOrCreate(ctx, chatID)
	if err != nil {
		return fmt.Errorf("get or create user: %w", err)
	}

	if err := s.history.Delete(ctx, userID); err != nil {
		return fmt.Errorf("delete history: %w", err)
	}

	return nil
}
