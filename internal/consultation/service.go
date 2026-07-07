package consultation

import (
	"context"
	"errors"
	"kubometr/internal/ai"
	"kubometr/internal/state"
	"strings"
	"sync"
	"time"
)

var ErrUnknownUserState = errors.New("unknown user state")

type Service struct{
	state           *state.StateManager
	ai              *ai.Client
	aiTimeout       time.Duration
	aiRateLimit     time.Duration
	maxPromptLength int
	aiLimiter       chan struct{}
	mu              sync.Mutex
	lastAIRequest   map[int64]time.Time
}

func New(
	state *state.StateManager, 
	ai *ai.Client, 
	aiTimeout time.Duration, 
	aiRateLimit time.Duration, 
	maxPromptLength int, 
	maxConcurrentAI int,
) *Service {
	return &Service{
		state: state,
		ai: ai,
		aiTimeout:	aiTimeout,
		aiRateLimit: aiRateLimit,
		maxPromptLength: maxPromptLength,
		aiLimiter: make(chan struct{}, maxConcurrentAI),
		lastAIRequest: make(map[int64]time.Time),
	}
}

func (s *Service) Process(ctx context.Context, chatID int64, question string) (string, error){

	userState := s.state.Get(chatID)
	
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

		aiCtx, cancel := context.WithTimeout(ctx, s.aiTimeout)
		defer cancel()

		promt := buildPrompt(question)
		
		answer, err := s.ai.Ask(aiCtx, promt)	
		if err != nil{
			return  "", err
		}
		 
		answer = strings.TrimSpace(answer)
		if answer == "" {
			return "AI-консультант вернул пустой ответ. Попробуйте переформулировать вопрос.", nil
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

func (s *Service) Start(chatID int64){
	s.state.Set(chatID, state.StateConsultation)
}

func (s *Service) Reset(chatID int64){
	s.state.Delete(chatID)
}