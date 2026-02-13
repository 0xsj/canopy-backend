package database

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// IsNotFound reports whether the error is a pgx "no rows" error.
func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

// IsUniqueViolation reports whether the error is a Postgres unique constraint violation (23505).
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// IsFKViolation reports whether the error is a Postgres foreign key violation (23503).
func IsFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

// MapQueryError converts a pgx error into a canopy error with operation context.
func MapQueryError(err error, op string) error {
	if err == nil {
		return nil
	}
	if IsNotFound(err) {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	if IsUniqueViolation(err) || IsFKViolation(err) {
		return canopyerr.Wrap(canopyerr.ErrConflict, op)
	}
	return canopyerr.Wrap(canopyerr.ErrInternal, op).WithCause(err)
}

// StringsFromIDs converts a slice of typed IDs to a slice of strings.
func StringsFromIDs[T any](ids []types.ID[T]) []string {
	if ids == nil {
		return nil
	}
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = id.String()
	}
	return strs
}

// IDsFromStrings converts a slice of strings to a slice of typed IDs.
func IDsFromStrings[T any](strs []string) []types.ID[T] {
	if strs == nil {
		return nil
	}
	ids := make([]types.ID[T], len(strs))
	for i, s := range strs {
		ids[i] = types.IDFrom[T](s)
	}
	return ids
}
