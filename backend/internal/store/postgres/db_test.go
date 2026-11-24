package postgres

import (
	"context"
	"os"
	"testing"
	"time"
)

// Integration test: needs a throwaway database.
//
//	TEST_DATABASE_URL=postgres://cine:test@localhost:5433/cine?sslmode=disable go test ./internal/store/...
func testDatabaseURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}
	return url
}

func TestConnectAndMigrate_IsIdempotent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := Connect(ctx, testDatabaseURL(t))
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer pool.Close()

	for i := 1; i <= 2; i++ {
		if err := Migrate(ctx, pool); err != nil {
			t.Fatalf("Migrate run %d: %v", i, err)
		}
	}

	var showtimes int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM showtimes").Scan(&showtimes); err != nil {
		t.Fatalf("query: %v", err)
	}
	if showtimes != 56 {
		t.Errorf("showtimes = %d, want 56 (seed applied exactly once)", showtimes)
	}

	var version int64
	if err := pool.QueryRow(ctx, "SELECT max(version_id) FROM goose_db_version").Scan(&version); err != nil {
		t.Fatalf("query goose version: %v", err)
	}
	if version != 2 {
		t.Errorf("goose version = %d, want 2", version)
	}
}

func TestConnect_FailsFastOnUnreachableDatabase(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if _, err := Connect(ctx, "postgres://nobody:x@127.0.0.1:1/none?sslmode=disable&connect_timeout=1"); err == nil {
		t.Fatal("expected error for unreachable database")
	}
}

func TestConnect_RejectsInvalidURL(t *testing.T) {
	if _, err := Connect(context.Background(), "::not a url::"); err == nil {
		t.Fatal("expected error for invalid URL")
	}
}
