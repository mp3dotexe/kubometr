package consultation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"kubometr/internal/history"
	"kubometr/internal/state"
)

var ErrUnknownUserState = errors.New("unknown user state")

const (
	historyLimit = 20

	// rateLimitPruneThreshold bounds the lastAIRequest map: once it grows past
	// this size, entries older than the rate limit window are dropped.
	rateLimitPruneThreshold = 10_000
)

type Service struct {
	state           stateStore
	ai              aiCompleter
	history         historyStore
	users           userStore
	aiTimeout       time.Duration
	aiRateLimit     time.Duration
	maxPromptLength int
	aiLimiter       chan struct{}
	mu              sync.Mutex
	lastAIRequest   map[int64]time.Time
}

func New(
	state stateStore,
	ai aiCompleter,
	aiTimeout time.Duration,
	aiRateLimit time.Duration,
	maxPromptLength int,
	maxConcurrentAI int,
	history historyStore,
	users userStore,
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

func (s *Service) Process(ctx context.Context, id int64, question string) (string, error) {
	switch s.state.Get(id) {
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

		if !s.canAskAI(id, time.Now()) {
			return "Пожалуйста, подождите несколько секунд перед следующим вопросом.", nil
		}

		select {
		case s.aiLimiter <- struct{}{}:
			defer func() { <-s.aiLimiter }()
		default:
			return "Сейчас много запросов. Попробуйте еще раз через минуту.", nil
		}

		return s.ask(ctx, id, question)
	}

	return "", ErrUnknownUserState
}

func (s *Service) ask(ctx context.Context, id int64, question string) (string, error) {
	userID, err := s.users.GetOrCreate(ctx, id)
	if err != nil {
		return "", fmt.Errorf("get or create user: %w", err)
	}

	if err := s.history.Save(ctx, userID, string(history.UserRole), question); err != nil {
		return "", fmt.Errorf("save user message: %w", err)
	}

	messages, err := s.history.LoadHistory(ctx, userID, historyLimit)
	if err != nil {
		return "", fmt.Errorf("load history: %w", err)
	}

	aiCtx, cancel := context.WithTimeout(ctx, s.aiTimeout)
	defer cancel()

	answer, err := s.ai.Complete(aiCtx, buildMessages(messages))
	if err != nil {
		return "", fmt.Errorf("ask ai: %w", err)
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

func (s *Service) canAskAI(id int64, now time.Time) bool {
	if s.aiRateLimit <= 0 {
		return true
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	lastRequestAt, ok := s.lastAIRequest[id]
	if ok && now.Sub(lastRequestAt) < s.aiRateLimit {
		return false
	}

	if len(s.lastAIRequest) >= rateLimitPruneThreshold {
		for key, at := range s.lastAIRequest {
			if now.Sub(at) >= s.aiRateLimit {
				delete(s.lastAIRequest, key)
			}
		}
	}

	s.lastAIRequest[id] = now
	return true
}

func (s *Service) Start(id int64) {
	s.state.Set(id, state.StateConsultation)
}

func (s *Service) Reset(ctx context.Context, id int64) error {
	s.state.Delete(id)

	s.mu.Lock()
	delete(s.lastAIRequest, id)
	s.mu.Unlock()

	userID, err := s.users.GetOrCreate(ctx, id)
	if err != nil {
		return fmt.Errorf("get or create user: %w", err)
	}

	if err := s.history.Delete(ctx, userID); err != nil {
		return fmt.Errorf("delete history: %w", err)
	}

	return nil
}
