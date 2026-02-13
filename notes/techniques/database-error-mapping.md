# Database Error Mapping

## What

A single helper function `MapQueryError` in `pkg/database/pghelpers.go` translates PostgreSQL-specific errors into domain error kinds. Every adapter method calls it instead of handling pg errors directly.

## Why

- Adapters don't leak pg-specific error types into the domain or service layer
- Consistent error classification — `ErrNotFound`, `ErrConflict`, `ErrInternal` — across all repositories
- Service code can check `canopyerr.GetKind(err)` without importing pgx or knowing about SQL error codes
- One place to update if the error mapping rules change

## Example

```go
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

func IsNotFound(err error) bool {
    return errors.Is(err, pgx.ErrNoRows)
}

func IsUniqueViolation(err error) bool {
    var pgErr *pgconn.PgError
    return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func IsFKViolation(err error) bool {
    var pgErr *pgconn.PgError
    return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
```

## Usage in Adapters

```go
func (r *OrgRepository) Create(ctx context.Context, org domain.Organization) error {
    const op = "organization: create org"
    params, err := orgToCreateParams(org)
    if err != nil {
        return database.MapQueryError(err, op)
    }
    return database.MapQueryError(r.q.CreateOrg(ctx, params), op)
}

func (r *OrgRepository) FindByID(ctx context.Context, id types.OrgID) (domain.Organization, error) {
    const op = "organization: find by id"
    row, err := r.q.FindOrgByID(ctx, id.String())
    if err != nil {
        return domain.Organization{}, database.MapQueryError(err, op)
    }
    return orgToDomain(row)
}
```

## RowsAffected Pattern

For UPDATE and DELETE operations where pgx doesn't return `ErrNoRows`:

```go
func (r *MemberRepository) Remove(ctx context.Context, orgID types.OrgID, userID types.UserID) error {
    const op = "organization: remove member"
    result, err := r.q.RemoveMember(ctx, sqlc.RemoveMemberParams{...})
    if err != nil {
        return database.MapQueryError(err, op)
    }
    if result.RowsAffected() == 0 {
        return canopyerr.Wrap(canopyerr.ErrNotFound, op)
    }
    return nil
}
```

## Error Flow

```
PostgreSQL → pgx error → MapQueryError → canopyerr sentinel → Wrap(err, op) in service → HTTP handler
   pgx.ErrNoRows          → ErrNotFound  (KindNotFound)       → 404
   unique violation 23505  → ErrConflict  (KindConflict)       → 409
   FK violation 23503      → ErrConflict  (KindConflict)       → 409
   anything else           → ErrInternal  (KindInternal)       → 500
```

## Gotchas

- Always pass the `const op` string — it appears in error traces and is critical for debugging
- `MapQueryError` with a nil error returns nil — safe to call unconditionally
- The `RowsAffected()` check is needed for UPDATE/DELETE because pgx returns no error when zero rows match
- Don't double-map: if `orgToCreateParams` returns an error, that's a marshaling error (internal), not a pg error — but `MapQueryError` still wraps it correctly as internal

## Related

- [[sqlc-mapper-pattern]] — mappers convert between domain and sqlc types; repos call MapQueryError
- [[error-chain-metadata]] — MapQueryError produces errors that carry kind, code, and severity
- [[builder-pattern-errors]] — ErrNotFound, ErrConflict, ErrInternal are built with the fluent builder
