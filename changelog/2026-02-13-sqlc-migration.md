# 2026-02-13 — Migrate All Repository Adapters to sqlc + Mappers

## What was done
- Promoted shared helpers to `pkg/database/pghelpers.go`: `NullableString`, `DerefString`, `TsUpdatedAt`
- Added `sqlc-generate` Makefile target for all 12 contexts
- Created `sqlc.yaml`, `queries.sql`, `mapper.go` for all 12 bounded contexts
- Generated sqlc code (48 generated files: db.go, models.go, querier.go, queries.sql.go per context)
- Rewrote all 19 repository adapters as thin wrappers over sqlc queriers + mappers
- Removed all local per-file helpers (nullableString, derefString, tsUpdatedAt, rowScanner)
- All old inline SQL strings, manual row.Scan, and query builder code replaced

### Build order
- **Wave 1** (pattern setters): identity, discussion, seed
- **Wave 2** (moderate): deliverable, synthesis, session, notification
- **Wave 3** (multi-repo + JOINs): workspace, organization, convergence
- **Wave 4** (dynamic filters): exploration (sqlc.narg for LeafFilter), ledger (sqlc.narg for SystemFilter + DomainFilter)

### Verification
- `go build ./...` — clean
- `go vet ./...` — clean
- `go test ./...` — all 113 domain tests passing

## Decisions made
- Chose `json.RawMessage` over `[]byte` for JSONB (sqlc rejects bare `[]byte`)
- Used `sqlc.narg()` for optional filter parameters instead of dynamic WHERE building
- Added nullable `timestamptz → *time.Time` override for ledger context
- Used `sqlc.arg('name')::text` to fix scalar-vs-array inference in ANY() queries
- Generated code committed to repo (not gitignored) — matches sqlc best practices
- Created ADR-007 documenting the sqlc decision

## Open questions
- Integration tests for adapters not yet written (no Postgres in CI yet)
- `Leafe` model name (sqlc singularization of "leaves") is awkward but functional
- Graph query engine (exploration) still uses raw SQL — not migrated to sqlc (recursive CTEs + AGE)

## Next steps
- Service layer for each bounded context (application logic, event publishing)
- HTTP handlers
- Composition root wiring in `cmd/server/`
