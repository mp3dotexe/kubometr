package consultation

import (
	"strings"
	"testing"

	"kubometr/internal/ai"
	"kubometr/internal/history"
)

func TestBuildMessagesFirstMessage(t *testing.T) {
	got := buildMessages([]history.Message{
		{Role: history.UserRole, Text: "Нужна краска для ванной"},
	})

	if len(got) != 2 {
		t.Fatalf("got %d messages, want 2", len(got))
	}
	if got[0].Role != ai.RoleSystem || strings.Contains(got[0].Content, continuationNote) {
		t.Fatalf("system message = %+v, want prompt without continuation note", got[0])
	}
	if got[1] != (ai.Message{Role: ai.RoleUser, Content: "Нужна краска для ванной"}) {
		t.Fatalf("user message = %+v", got[1])
	}
}

func TestBuildMessagesKeepsRolesAndOrder(t *testing.T) {
	got := buildMessages([]history.Message{
		{Role: history.UserRole, Text: "Утеплить балкон"},
		{Role: history.AIRole, Text: "Какая площадь?"},
		{Role: history.UserRole, Text: "6 м²"},
	})

	want := []ai.Role{ai.RoleSystem, ai.RoleUser, ai.RoleAssistant, ai.RoleUser}
	if len(got) != len(want) {
		t.Fatalf("got %d messages, want %d", len(got), len(want))
	}
	for i, role := range want {
		if got[i].Role != role {
			t.Fatalf("message %d role = %q, want %q", i, got[i].Role, role)
		}
	}
	if !strings.Contains(got[0].Content, continuationNote) {
		t.Fatal("system message has no continuation note for an ongoing dialog")
	}
	if got[3].Content != "6 м²" {
		t.Fatalf("last message = %q", got[3].Content)
	}
}
