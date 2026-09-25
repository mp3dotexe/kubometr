package database

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"

	"kubometr/internal/config"
	"kubometr/internal/testdb"
	"kubometr/migrations"
)

func TestLoadMigrationsSortsAndFilters(t *testing.T) {
	fsys := fstest.MapFS{
		"000002_second.up.sql":   {Data: []byte("SELECT 2")},
		"000002_second.down.sql": {Data: []byte("SELECT -2")},
		"000001_first.up.sql":    {Data: []byte("SELECT 1")},
		"embed.go":               {Data: []byte("package migrations")},
	}

	got, err := loadMigrations(fsys)
	if err != nil {
		t.Fatalf("loadMigrations() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d migrations, want 2", len(got))
	}
	if got[0].version != 1 || got[1].version != 2 {
		t.Fatalf("versions = %d, %d; want 1, 2", got[0].version, got[1].version)
	}
	if got[1].sql != "SELECT 2" {
		t.Fatalf("sql = %q", got[1].sql)
	}
}

func TestLoadMigrationsRejectsDuplicates(t *testing.T) {
	fsys := fstest.MapFS{
		"000001_a.up.sql": {Data: []byte("SELECT 1")},
		"000001_b.up.sql": {Data: []byte("SELECT 1")},
	}

	if _, err := loadMigrations(fsys); err == nil {
		t.Fatal("loadMigrations() error = nil, want duplicate error")
	}
}

func TestEmbeddedMigrationsLoad(t *testing.T) {
	got, err := loadMigrations(migrations.FS)
	if err != nil {
		t.Fatalf("loadMigrations() error = %v", err)
	}
	if len(got) == 0 {
		t.Fatal("no embedded migrations found")
	}
}

func TestDSNEscapesCredentials(t *testing.T) {
	got := dsn(&config.Config{
		PostgresUser:     "app",
		PostgresPassword: "p@ss/word",
		PostgresHost:     "localhost",
		PostgresPort:     5432,
		PostgresDB:       "kubometr_db",
	})

	want := "postgres://app:p%40ss%2Fword@localhost:5432/kubometr_db"
	if got != want {
		t.Fatalf("dsn() = %q, want %q", got, want)
	}
}

func TestMigrateAppliesAllAndIsIdempotent(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()

	if err := Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("first Migrate() error = %v", err)
	}
	if err := Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("second Migrate() error = %v", err)
	}

	var version int64
	if err := pool.QueryRow(ctx, "SELECT version FROM schema_migrations").Scan(&version); err != nil {
		t.Fatalf("read version: %v", err)
	}
	if version != 2 {
		t.Fatalf("version = %d, want 2", version)
	}

	// Same external ID on different platforms must be two different users.
	_, err := pool.Exec(ctx, `INSERT INTO users (platform, external_id) VALUES ('telegram', 1), ('max', 1)`)
	if err != nil {
		t.Fatalf("insert users: %v", err)
	}
}

// A database created by hand from 000001 before the migrator existed has
// the tables but no schema_migrations; the migrator must adopt it.
func TestMigrateAdoptsManuallyCreatedSchema(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()

	initSQL, err := migrations.FS.ReadFile("000001_init.up.sql")
	if err != nil {
		t.Fatalf("read init migration: %v", err)
	}
	if _, err := pool.Exec(ctx, string(initSQL)); err != nil {
		t.Fatalf("apply init by hand: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO users (telegram_id) VALUES (42)`); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	if err := Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	var platform string
	err = pool.QueryRow(ctx, `SELECT platform FROM users WHERE external_id = 42`).Scan(&platform)
	if err != nil {
		t.Fatalf("select migrated user: %v", err)
	}
	if platform != "telegram" {
		t.Fatalf("platform = %q, want telegram", platform)
	}
}

func TestMigrateRejectsDirtySchema(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()

	_, err := pool.Exec(ctx, `
		CREATE TABLE schema_migrations (version BIGINT NOT NULL PRIMARY KEY, dirty BOOLEAN NOT NULL);
		INSERT INTO schema_migrations VALUES (1, true);`)
	if err != nil {
		t.Fatalf("prepare dirty schema: %v", err)
	}

	err = Migrate(ctx, pool, migrations.FS)
	if err == nil || !strings.Contains(err.Error(), "dirty") {
		t.Fatalf("Migrate() error = %v, want dirty error", err)
	}
}
