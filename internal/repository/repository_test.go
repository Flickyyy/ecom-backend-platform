package repository

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		fmt.Println("TEST_DATABASE_URL not set, skipping integration tests")
		os.Exit(0)
	}

	var err error
	testPool, err = pgxpool.New(context.Background(), dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to test database: %v\n", err)
		os.Exit(1)
	}
	defer testPool.Close()

	code := m.Run()
	os.Exit(code)
}

func cleanupTable(t *testing.T, tables ...string) {
	t.Helper()
	for _, table := range tables {
		_, err := testPool.Exec(context.Background(), fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			t.Fatalf("failed to cleanup table %s: %v", table, err)
		}
	}
}
