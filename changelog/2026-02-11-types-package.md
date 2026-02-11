# 2026-02-11 — Types Package

## What was done
- Built `pkg/types` — shared domain types used across bounded contexts
- Implemented: typed IDs with phantom type parameters, timestamps (immutable + mutable), cursor-based pagination, discriminated union API response envelope
- Response envelope supports three variants: OK (data), Fail (error), FailWithDetails (validation with field-level errors)
- Updated `cmd/demo/main.go` to exercise all packages together — typed IDs, timestamps, pagination, all three response shapes
- Full test coverage (id, timestamp, pagination, response including HTTP write helpers)

## Decisions made
- Generic `ID[T]` with phantom type tags — prevents mixing `LeafID` with `UserID` at compile time
- Prefixed ID format (`leaf_a1b2c3...`) — human-readable, self-describing
- `ParseID` validates prefix for untrusted input, `IDFrom` trusts database values
- Timestamps always normalized to UTC — no timezone ambiguity
- Immutable entities use `NewTimestamps()` (no UpdatedAt), mutable use `NewMutableTimestamps()`
- Cursor-based pagination over offset — better for graph/timeline views, stable under inserts
- Cursors are base64-encoded to keep internal details opaque from clients
- Response is a discriminated union with `Validate()` to catch invalid states (both set, neither set)
- `FieldError` struct for validation responses — frontend can map errors to specific form fields
- `WriteValidationError` standardizes the 400 response shape across all handlers

## Open questions
- None for types package

## Next steps
- Begin domain types for individual bounded contexts, or build database/NATS infrastructure packages
