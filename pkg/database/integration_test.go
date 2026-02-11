//go:build integration

package database

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5"

	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

const testDSN = "postgres://canopy:canopy@localhost:5433/canopy?sslmode=disable"

func testDB(t *testing.T) *DB {
	t.Helper()
	ctx := context.Background()
	cfg := Config{
		DSN:         testDSN,
		MaxPoolSize: 5,
		MinPoolSize: 1,
	}
	db, err := Connect(ctx, cfg, logger.NewNoop())
	if err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// --- Connect ---

func TestConnect_Success(t *testing.T) {
	db := testDB(t)
	if db.Pool() == nil {
		t.Fatal("Pool() is nil after Connect")
	}
}

func TestConnect_InvalidDSN(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		DSN:         "postgres://nobody:wrong@localhost:59999/nope",
		MaxPoolSize: 5,
		MinPoolSize: 1,
	}
	_, err := Connect(ctx, cfg, logger.NewNoop())
	if err == nil {
		t.Fatal("Connect() = nil, want error for bad DSN")
	}
}

// --- Health ---

func TestHealth_Success(t *testing.T) {
	db := testDB(t)
	if err := db.Health(context.Background()); err != nil {
		t.Errorf("Health() = %v, want nil", err)
	}
}

// --- Stats ---

func TestStats_ReturnsPoolInfo(t *testing.T) {
	db := testDB(t)
	stats := db.Stats()
	if stats.TotalConns <= 0 {
		t.Errorf("TotalConns = %d, want > 0", stats.TotalConns)
	}
}

// --- DBTX ---

func TestDBTX_ReturnsPool(t *testing.T) {
	db := testDB(t)
	dbtx := db.DBTX()
	if dbtx == nil {
		t.Fatal("DBTX() returned nil")
	}

	// Verify it can execute a query.
	var result int
	err := dbtx.QueryRow(context.Background(), "SELECT 1").Scan(&result)
	if err != nil {
		t.Errorf("QueryRow via DBTX: %v", err)
	}
	if result != 1 {
		t.Errorf("result = %d, want 1", result)
	}
}

// --- WithTx ---

func TestWithTx_Commit(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	table := "test_withtx_commit"
	db.Pool().Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
	db.Pool().Exec(ctx, fmt.Sprintf("CREATE TABLE %s (id INT)", table))
	t.Cleanup(func() {
		db.Pool().Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
	})

	err := db.WithTx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, fmt.Sprintf("INSERT INTO %s (id) VALUES (42)", table))
		return err
	})
	if err != nil {
		t.Fatalf("WithTx() = %v, want nil", err)
	}

	var count int
	db.Pool().QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE id = 42", table)).Scan(&count)
	if count != 1 {
		t.Errorf("count = %d, want 1 (transaction should have committed)", count)
	}
}

func TestWithTx_Rollback(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	table := "test_withtx_rollback"
	db.Pool().Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
	db.Pool().Exec(ctx, fmt.Sprintf("CREATE TABLE %s (id INT)", table))
	t.Cleanup(func() {
		db.Pool().Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
	})

	testErr := errors.New("intentional failure")
	err := db.WithTx(ctx, func(tx pgx.Tx) error {
		tx.Exec(ctx, fmt.Sprintf("INSERT INTO %s (id) VALUES (99)", table))
		return testErr
	})

	if !errors.Is(err, testErr) {
		t.Fatalf("WithTx() = %v, want %v", err, testErr)
	}

	var count int
	db.Pool().QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE id = 99", table)).Scan(&count)
	if count != 0 {
		t.Errorf("count = %d, want 0 (transaction should have rolled back)", count)
	}
}

func TestWithTx_PanicRollback(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	table := "test_withtx_panic"
	db.Pool().Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
	db.Pool().Exec(ctx, fmt.Sprintf("CREATE TABLE %s (id INT)", table))
	t.Cleanup(func() {
		db.Pool().Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
	})

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic to propagate")
		}

		var count int
		db.Pool().QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if count != 0 {
			t.Errorf("count = %d, want 0 (panic should have triggered rollback)", count)
		}
	}()

	db.WithTx(ctx, func(tx pgx.Tx) error {
		tx.Exec(ctx, fmt.Sprintf("INSERT INTO %s (id) VALUES (1)", table))
		panic("test panic")
	})
}

// --- Migration ---

func TestMigrator_Run(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	schema := "test_migration"
	db.Pool().Exec(ctx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schema))
	db.Pool().Exec(ctx, fmt.Sprintf("CREATE SCHEMA %s", schema))
	t.Cleanup(func() {
		db.Pool().Exec(ctx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schema))
	})

	migrations := fstest.MapFS{
		"001_create_items.sql": {Data: []byte(`
			CREATE TABLE items (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL
			);
		`)},
		"002_add_status.sql": {Data: []byte(`
			ALTER TABLE items ADD COLUMN status TEXT DEFAULT 'active';
		`)},
	}

	m := NewMigrator(db, schema, logger.NewNoop())
	if err := m.Run(ctx, migrations); err != nil {
		t.Fatalf("Run() = %v, want nil", err)
	}

	// Verify table was created.
	var exists bool
	db.Pool().QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = $1 AND table_name = 'items'
		)`, schema,
	).Scan(&exists)
	if !exists {
		t.Error("items table not created")
	}

	// Verify both migrations recorded.
	var count int
	db.Pool().QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM %s.schema_migrations", schema),
	).Scan(&count)
	if count != 2 {
		t.Errorf("applied migrations = %d, want 2", count)
	}

	// Running again should be a no-op.
	if err := m.Run(ctx, migrations); err != nil {
		t.Fatalf("Run() second time = %v, want nil (idempotent)", err)
	}

	db.Pool().QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM %s.schema_migrations", schema),
	).Scan(&count)
	if count != 2 {
		t.Errorf("applied migrations after rerun = %d, want 2 (idempotent)", count)
	}
}

// --- Graph ---
//
// Graph tests require AGE to support LOAD 'age' without hanging.
// Some AGE Docker images have issues with LOAD — run these tests
// only when AGE_TESTS=1 is set and a working AGE instance is available.

func TestGraph_Exec_And_Query(t *testing.T) {
	if os.Getenv("AGE_TESTS") != "1" {
		t.Skip("set AGE_TESTS=1 to run graph tests (requires working AGE instance)")
	}

	db := testDB(t)
	ctx := context.Background()

	graph := NewGraph(db, "canopy")

	// Create a test vertex.
	err := graph.Exec(ctx, "CREATE (t:TestNode {name: 'integration_test'})")
	if err != nil {
		t.Fatalf("Graph.Exec(CREATE) = %v", err)
	}

	// Query it back.
	rows, err := graph.Query(ctx,
		"MATCH (t:TestNode {name: 'integration_test'}) RETURN t",
		"(node agtype)",
	)
	if err != nil {
		t.Fatalf("Graph.Query(MATCH) = %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Error("expected at least one row from graph query")
	}

	// Clean up.
	graph.Exec(ctx, "MATCH (t:TestNode {name: 'integration_test'}) DELETE t")
}
