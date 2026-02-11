package database

import (
	"testing"
	"testing/fstest"

	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

func TestMigrator_ReadMigrations_SortsLexicographically(t *testing.T) {
	migrations := fstest.MapFS{
		"003_add_index.sql":    {Data: []byte("CREATE INDEX ...")},
		"001_create_table.sql": {Data: []byte("CREATE TABLE ...")},
		"002_add_column.sql":   {Data: []byte("ALTER TABLE ...")},
		"readme.txt":           {Data: []byte("not a migration")},
	}

	m := &Migrator{log: logger.NewNoop()}
	files, err := m.readMigrations(migrations)
	if err != nil {
		t.Fatalf("readMigrations() error: %v", err)
	}

	if len(files) != 3 {
		t.Fatalf("got %d files, want 3 (should skip non-.sql)", len(files))
	}

	expected := []struct {
		name    string
		version string
	}{
		{"001_create_table.sql", "001"},
		{"002_add_column.sql", "002"},
		{"003_add_index.sql", "003"},
	}

	for i, want := range expected {
		if files[i].name != want.name {
			t.Errorf("files[%d].name = %q, want %q", i, files[i].name, want.name)
		}
		if files[i].version != want.version {
			t.Errorf("files[%d].version = %q, want %q", i, files[i].version, want.version)
		}
	}
}

func TestMigrator_ReadMigrations_SkipsDirectories(t *testing.T) {
	migrations := fstest.MapFS{
		"001_create.sql":        {Data: []byte("CREATE TABLE ...")},
		"subdir/002_nested.sql": {Data: []byte("should be skipped")},
	}

	m := &Migrator{log: logger.NewNoop()}
	files, err := m.readMigrations(migrations)
	if err != nil {
		t.Fatalf("readMigrations() error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("got %d files, want 1 (should skip subdirectories)", len(files))
	}

	if files[0].name != "001_create.sql" {
		t.Errorf("files[0].name = %q, want %q", files[0].name, "001_create.sql")
	}
}

func TestMigrator_ReadMigrations_Empty(t *testing.T) {
	migrations := fstest.MapFS{}

	m := &Migrator{log: logger.NewNoop()}
	files, err := m.readMigrations(migrations)
	if err != nil {
		t.Fatalf("readMigrations() error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("got %d files, want 0", len(files))
	}
}

func TestMigrator_ReadMigrations_NonexistentDir(t *testing.T) {
	// An empty MapFS returns no files from ReadDir.
	empty := fstest.MapFS{}

	m := &Migrator{log: logger.NewNoop()}
	files, err := m.readMigrations(empty)
	if err != nil {
		t.Fatalf("readMigrations(empty) = %v, want nil", err)
	}
	if len(files) != 0 {
		t.Errorf("got %d files from empty FS, want 0", len(files))
	}
}

func TestMigrator_ReadMigrations_PreservesSQL(t *testing.T) {
	sql := "CREATE TABLE leaves (id TEXT PRIMARY KEY);"
	migrations := fstest.MapFS{
		"001_create_leaves.sql": {Data: []byte(sql)},
	}

	m := &Migrator{log: logger.NewNoop()}
	files, err := m.readMigrations(migrations)
	if err != nil {
		t.Fatalf("readMigrations() error: %v", err)
	}

	if files[0].sql != sql {
		t.Errorf("sql = %q, want %q", files[0].sql, sql)
	}
}

func TestMigrator_ReadMigrations_VersionExtraction(t *testing.T) {
	tests := []struct {
		filename string
		version  string
	}{
		{"001_create_table.sql", "001"},
		{"v2_add_column.sql", "v2"},
		{"20260211_initial.sql", "20260211"},
	}

	for _, tt := range tests {
		migrations := fstest.MapFS{
			tt.filename: {Data: []byte("SELECT 1;")},
		}

		m := &Migrator{log: logger.NewNoop()}
		files, err := m.readMigrations(migrations)
		if err != nil {
			t.Fatalf("readMigrations() error for %s: %v", tt.filename, err)
		}

		if files[0].version != tt.version {
			t.Errorf("version for %s = %q, want %q", tt.filename, files[0].version, tt.version)
		}
	}
}
