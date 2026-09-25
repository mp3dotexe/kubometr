package history

import (
	"context"
	"testing"

	"kubometr/internal/chat"
	"kubometr/internal/database"
	"kubometr/internal/testdb"
	"kubometr/internal/users"
	"kubometr/migrations"
)

func TestRepositorySaveLoadDelete(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	if err := database.Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	userID, err := users.New(pool).GetOrCreate(ctx, chat.ID{Platform: chat.Telegram, ChatID: 1})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	repo := New(pool)
	texts := []string{"первый", "второй", "третий"}
	for i, text := range texts {
		role := UserRole
		if i%2 == 1 {
			role = AIRole
		}
		if err := repo.Save(ctx, userID, string(role), text); err != nil {
			t.Fatalf("Save(%q) error = %v", text, err)
		}
	}

	// Limit keeps the latest messages, returned oldest first.
	got, err := repo.LoadHistory(ctx, userID, 2)
	if err != nil {
		t.Fatalf("LoadHistory() error = %v", err)
	}
	if len(got) != 2 || got[0].Text != "второй" || got[1].Text != "третий" {
		t.Fatalf("LoadHistory() = %+v, want [второй третий]", got)
	}
	if got[0].Role != AIRole || got[1].Role != UserRole {
		t.Fatalf("roles = %q, %q", got[0].Role, got[1].Role)
	}

	if err := repo.Delete(ctx, userID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	got, err = repo.LoadHistory(ctx, userID, 10)
	if err != nil {
		t.Fatalf("LoadHistory() after Delete error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("history after Delete = %+v, want empty", got)
	}
}
