# sqlc + Mapper Pattern

## What
A three-layer adapter structure where sqlc generates type-safe query functions and model structs from SQL, a mapper layer converts between sqlc models and domain entities, and the repository becomes a thin wrapper that wires querier → error mapping → mapper.

## Why
Raw pgx queries mix SQL strings, scan logic, and domain reconstruction in a single file. This pattern separates concerns: SQL lives in `queries.sql` (validated at generation time), type conversions live in `mapper.go`, and the repository holds only orchestration logic. Adding a column means updating one SQL file and one mapper function — the repository doesn't change.

## Example

### queries.sql
```sql
-- name: FindUserByID :one
SELECT id, display_name, email, avatar_url, created_at, updated_at
FROM users WHERE id = $1;

-- name: UpdateUser :execresult
UPDATE users SET display_name = $2, email = $3, updated_at = $4
WHERE id = $1;
```

### mapper.go
```go
func userToDomain(row sqlc.User) domain.User {
    updatedTS := types.TimestampFrom(row.UpdatedAt)
    return domain.ReconstructUser(
        types.UserIDFrom(row.ID),
        row.DisplayName,
        row.Email,
        database.DerefString(row.AvatarUrl), // *string → string
        types.Timestamps{
            CreatedAt: types.TimestampFrom(row.CreatedAt),
            UpdatedAt: &updatedTS,
        },
    )
}
```

### repository.go
```go
func (r *UserRepository) FindByID(ctx context.Context, id types.UserID) (domain.User, error) {
    const op = "identity: find user by id"
    row, err := r.q.FindUserByID(ctx, id.String())
    if err != nil {
        return domain.User{}, database.MapQueryError(err, op)
    }
    return userToDomain(row), nil
}
```

## Gotchas

### JSONB type override
sqlc rejects `go_type: "[]byte"` for JSONB. Use the object form:
```yaml
- db_type: "jsonb"
  go_type:
    import: "encoding/json"
    type: "RawMessage"
```

### Integer override doesn't always apply
`db_type: "integer"` with `go_type: "int"` works for LIMIT params but not for column types — sqlc still generates `int32`. Handle with casts in mapper: `int(row.Count)`.

### Nullable timestamptz for sqlc.narg
Default nullable timestamptz maps to `pgtype.Timestamptz`. Add explicit override:
```yaml
- db_type: "timestamptz"
  nullable: true
  go_type:
    type: "Time"
    import: "time"
    pointer: true
```

### Scalar in ANY() array check
`WHERE $1 = ANY(array_col)` — sqlc infers `$1` as `[]string` from the column type. Fix with explicit typing:
```sql
WHERE sqlc.arg('leaf_id')::text = ANY(leaf_ids)
```

### Table name singularization
sqlc singularizes table names for models. Table `leaves` → model `Leafe` (not `Leaf`). Use the generated name in mapper functions.

## Related
- [[dbtx-interface-pattern]] — DBTX compatibility between database.DBTX and sqlc.DBTX
- [[per-schema-migrations]] — migrations that sqlc reads for schema inference
