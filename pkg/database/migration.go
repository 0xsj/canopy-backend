package database

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// Migrator runs versioned SQL migrations for a specific schema.
// Each bounded context has its own Migrator scoped to its schema.
type Migrator struct {
	db     *DB
	schema string
	log    logger.Logger
}

// NewMigrator creates a migrator for the given schema.
func NewMigrator(db *DB, schema string, log logger.Logger) *Migrator {
	return &Migrator{db: db, schema: schema, log: log}
}

// Run applies all unapplied migrations from the given filesystem.
// Migration files must be named "{version}_{description}.sql" and
// are applied in lexicographic order.
//
// Example filesystem (embed with go:embed):
//
//	migrations/
//	  001_create_leaves.sql
//	  002_add_tags_column.sql
func (m *Migrator) Run(ctx context.Context, migrations fs.FS) error {
	if err := m.ensureSchema(ctx); err != nil {
		return err
	}
	if err := m.ensureTable(ctx); err != nil {
		return err
	}

	applied, err := m.appliedVersions(ctx)
	if err != nil {
		return err
	}

	files, err := m.readMigrations(migrations)
	if err != nil {
		return err
	}

	for _, mf := range files {
		if applied[mf.version] {
			continue
		}

		m.log.Info("applying migration",
			logger.String("schema", m.schema),
			logger.String("version", mf.version),
			logger.String("file", mf.name),
		)

		if err := m.apply(ctx, mf); err != nil {
			return fmt.Errorf("database: migration %s: %w", mf.name, err)
		}
	}

	return nil
}

type migrationFile struct {
	name    string
	version string
	sql     string
}

func (m *Migrator) ensureSchema(ctx context.Context) error {
	_, err := m.db.pool.Exec(ctx, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", m.schema))
	if err != nil {
		return fmt.Errorf("database: create schema %s: %w", m.schema, err)
	}
	return nil
}

func (m *Migrator) ensureTable(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`, m.schema)

	_, err := m.db.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("database: create migrations table in %s: %w", m.schema, err)
	}
	return nil
}

func (m *Migrator) appliedVersions(ctx context.Context) (map[string]bool, error) {
	query := fmt.Sprintf(`SELECT version FROM %s.schema_migrations`, m.schema)

	rows, err := m.db.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("database: query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("database: scan migration version: %w", err)
		}
		applied[version] = true
	}
	return applied, rows.Err()
}

func (m *Migrator) readMigrations(migrations fs.FS) ([]migrationFile, error) {
	entries, err := fs.ReadDir(migrations, ".")
	if err != nil {
		return nil, fmt.Errorf("database: read migration directory: %w", err)
	}

	var files []migrationFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		content, err := fs.ReadFile(migrations, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("database: read migration %s: %w", entry.Name(), err)
		}

		version := strings.SplitN(entry.Name(), "_", 2)[0]

		files = append(files, migrationFile{
			name:    entry.Name(),
			version: version,
			sql:     string(content),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].name < files[j].name
	})

	return files, nil
}

func (m *Migrator) apply(ctx context.Context, mf migrationFile) error {
	return m.db.WithTx(ctx, func(tx pgx.Tx) error {
		// Set the search path to this schema for the migration.
		if _, err := tx.Exec(ctx, fmt.Sprintf("SET search_path TO %s, public", m.schema)); err != nil {
			return fmt.Errorf("set search_path: %w", err)
		}

		// Execute the migration SQL.
		if _, err := tx.Exec(ctx, mf.sql); err != nil {
			return fmt.Errorf("execute: %w", err)
		}

		// Record the migration as applied.
		record := fmt.Sprintf(
			`INSERT INTO %s.schema_migrations (version) VALUES ($1)`,
			m.schema,
		)
		if _, err := tx.Exec(ctx, record, mf.version); err != nil {
			return fmt.Errorf("record version: %w", err)
		}

		return nil
	})
}

// MigrationSet collects schema-scoped migrations for batch execution.
// The composition root uses this to run all bounded-context migrations at startup.
type MigrationSet struct {
	entries []migrationEntry
}

type migrationEntry struct {
	schema     string
	migrations fs.FS
}

// NewMigrationSet creates an empty migration set.
func NewMigrationSet() *MigrationSet {
	return &MigrationSet{}
}

// Add registers migrations for a schema. Call once per bounded context.
func (ms *MigrationSet) Add(schema string, migrations fs.FS) {
	ms.entries = append(ms.entries, migrationEntry{schema: schema, migrations: migrations})
}

// RunAll applies all registered migrations in order. Each schema is migrated
// independently with its own Migrator instance.
func (ms *MigrationSet) RunAll(ctx context.Context, db *DB, log logger.Logger) error {
	for _, entry := range ms.entries {
		migrator := NewMigrator(db, entry.schema, log)
		if err := migrator.Run(ctx, entry.migrations); err != nil {
			return fmt.Errorf("database: migrate %s: %w", entry.schema, err)
		}
	}
	return nil
}
