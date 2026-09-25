package requests

import (
	"context"
	"testing"

	"kubometr/internal/chat"
	"kubometr/internal/database"
	"kubometr/internal/testdb"
	"kubometr/internal/users"
	"kubometr/migrations"
)

func TestRepositoryCreateListSetStatus(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	if err := database.Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	owner := chat.ID{Platform: chat.Telegram, ChatID: 7}
	userID, err := users.New(pool).GetOrCreate(ctx, owner)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	repo := NewRepository(pool)
	created, err := repo.Create(ctx, userID, Request{Phone: "+79991234567", Question: "• балкон", Answer: "пеноплекс"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID == 0 || created.Status != StatusNew || created.CreatedAt.IsZero() {
		t.Fatalf("created = %+v", created)
	}

	list, err := repo.List(ctx, userID, 10)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list) != 1 || list[0].ID != created.ID || list[0].Question != "• балкон" {
		t.Fatalf("List() = %+v", list)
	}

	updated, ok, err := repo.SetStatus(ctx, created.ID, StatusInProgress)
	if err != nil || !ok {
		t.Fatalf("SetStatus() = %v, %v", ok, err)
	}
	if updated.Status != StatusInProgress || updated.Client != owner {
		t.Fatalf("updated = %+v, want in_progress for %+v", updated, owner)
	}

	// The same status again is not a change, and neither is a missing request.
	if _, ok, err := repo.SetStatus(ctx, created.ID, StatusInProgress); err != nil || ok {
		t.Fatalf("repeated SetStatus() = %v, %v, want false", ok, err)
	}
	if _, ok, err := repo.SetStatus(ctx, created.ID+100, StatusDone); err != nil || ok {
		t.Fatalf("SetStatus(missing) = %v, %v, want false", ok, err)
	}
}
