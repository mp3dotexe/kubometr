package users

import (
	"context"
	"sync"
	"testing"

	"kubometr/internal/chat"
	"kubometr/internal/database"
	"kubometr/internal/testdb"
	"kubometr/migrations"
)

func newRepository(t *testing.T) *Repository {
	t.Helper()
	pool := testdb.New(t)
	if err := database.Migrate(context.Background(), pool, migrations.FS); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return New(pool)
}

func TestGetOrCreateReturnsSameUser(t *testing.T) {
	repo := newRepository(t)
	ctx := context.Background()
	id := chat.ID{Platform: chat.Telegram, ChatID: 100}

	first, err := repo.GetOrCreate(ctx, id)
	if err != nil {
		t.Fatalf("first GetOrCreate() error = %v", err)
	}
	second, err := repo.GetOrCreate(ctx, id)
	if err != nil {
		t.Fatalf("second GetOrCreate() error = %v", err)
	}
	if first != second {
		t.Fatalf("got different users %d and %d for the same chat", first, second)
	}
}

func TestGetOrCreateSeparatesPlatforms(t *testing.T) {
	repo := newRepository(t)
	ctx := context.Background()

	tg, err := repo.GetOrCreate(ctx, chat.ID{Platform: chat.Telegram, ChatID: 100})
	if err != nil {
		t.Fatalf("GetOrCreate(telegram) error = %v", err)
	}
	mx, err := repo.GetOrCreate(ctx, chat.ID{Platform: chat.MAX, ChatID: 100})
	if err != nil {
		t.Fatalf("GetOrCreate(max) error = %v", err)
	}
	if tg == mx {
		t.Fatal("telegram and MAX chats with the same ID share a user")
	}
}

func TestGetOrCreateConcurrent(t *testing.T) {
	repo := newRepository(t)
	id := chat.ID{Platform: chat.MAX, ChatID: 200}

	const workers = 10
	ids := make([]int64, workers)
	errs := make([]error, workers)

	var wg sync.WaitGroup
	for i := range workers {
		wg.Go(func() {
			ids[i], errs[i] = repo.GetOrCreate(context.Background(), id)
		})
	}
	wg.Wait()

	for i := range workers {
		if errs[i] != nil {
			t.Fatalf("worker %d error = %v", i, errs[i])
		}
		if ids[i] != ids[0] {
			t.Fatalf("worker %d got user %d, want %d", i, ids[i], ids[0])
		}
	}
}
