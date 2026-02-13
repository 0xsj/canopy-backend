package postgres

import (
	"embed"
	"io/fs"
)

//go:embed migrations/*.sql
var migrations embed.FS

// MigrationFS returns the migration filesystem for the workspace schema.
func MigrationFS() fs.FS {
	sub, err := fs.Sub(migrations, "migrations")
	if err != nil {
		panic("postgres: migrations sub-filesystem: " + err.Error())
	}
	return sub
}
