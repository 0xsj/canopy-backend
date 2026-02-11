# 2026-02-11 — Database Package

## What was done
- Built `pkg/database/database.go` — core `DB` wrapper around `pgxpool.Pool` with `Connect`, `Close`, `Pool`, and `DBTX` methods
- Defined the `DBTX` interface (`Exec`, `Query`, `QueryRow`) — satisfied by both `*pgxpool.Pool` and `pgx.Tx`, so sqlc-generated code works transparently inside or outside a transaction
- Implemented `Config` struct with `Load`/`Validate` methods — integrates with `pkg/config` as a `config.Section`, reads DSN, pool sizes, and connection lifetimes from environment variables
- Added `sanitizedDSN()` to mask passwords in log output
- Built `pkg/database/transaction.go` — `WithTx` runs a function inside a transaction with automatic commit/rollback, `WithTxOptions` adds custom isolation and access mode support
- Both transaction helpers handle panics (rollback then re-panic) and wrap rollback failures with the original error
- Built `pkg/database/migration.go` — `Migrator` struct scoped to a specific Postgres schema, runs versioned SQL migrations from an `fs.FS` (designed for `go:embed`)
- Migration system creates a `schema_migrations` tracking table per schema, reads `{version}_{description}.sql` files in lexicographic order, skips already-applied versions, runs each migration inside a transaction with the schema's search path set
- Built `pkg/database/health.go` — `Health` method with a 3-second timeout, `Stats` method exposing pool statistics (total, idle, acquired connections)
- Built `pkg/database/graph.go` — `Graph` helper wrapping Apache AGE boilerplate, provides `Query` (returns rows) and `Exec` (no rows) methods that accept Cypher queries, handles `LOAD 'age'` and search path setup per call via `ensureAGE`

## Decisions made
- `DBTX` as the universal database interface — adapters accept `DBTX` so the same repository code works with a pool connection or a transaction, no code changes needed to add transactional behavior
- Config implements `config.Section` — consistent with the project's config loading pattern, all database settings loaded and validated uniformly
- Per-schema migration tracking — each bounded context has its own `schema_migrations` table in its own Postgres schema, contexts migrate independently without coordination
- Migrations use `fs.FS` — designed for `go:embed` so migration SQL files are compiled into the binary, no filesystem dependency at runtime
- Migrations run inside transactions — a failed migration rolls back cleanly, no partial state
- `ensureAGE` called per graph operation — AGE requires `LOAD 'age'` and search path setup per connection, so this is done before every query to be safe with pooled connections
- Graph helper accepts raw Cypher and a column type clause — keeps the abstraction thin, domain code writes Cypher without SQL boilerplate but retains full control over query shape

## Open questions
- `ensureAGE` runs `LOAD 'age'` before every graph call — is there a way to set this once per connection via pgxpool's `AfterConnect` hook to avoid the repeated overhead?
- Graph helper currently uses the pool directly — should it also accept `DBTX` so graph mutations can participate in the same transaction as relational writes?
- No retry logic on transient connection failures — is that handled at a higher layer or should the database package include it?

## Next steps
- Build `pkg/nats` package for NATS/JetStream connectivity
- Begin domain types and ports for the first bounded context
- Wire `pkg/database` into the composition root (`cmd/server/`)
