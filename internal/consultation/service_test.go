package consultation
import (
	"testing"
	"time"
)

func TestCanAskAI_RateLimit(t *testing.T) {
	s := &Service{
		aiRateLimit: 3 * time.Second,
		lastAIRequest: make(map[int64]time.Time),
	}

	now := time.Now()

	allowed := s.canAskAI(1, now)
	if !allowed {
		t.Errorf("expected true on first call, got false")
	}

	allowed = s.canAskAI(1, now)
	if allowed {
		t.Errorf("expected false on second call, got true")
	}

	allowed = s.canAskAI(2, now)
	if !allowed {
		t.Errorf("expected true on third call, got false")
	}

	later := now.Add(4 * time.Second)
	allowed = s.canAskAI(1, later)
	if !allowed {
		t.Errorf("expected true on fourth call, got false")
	}
}
