# 2026-02-14 — Incomplete Flows: Convergence, Deliverable, Notification

## What was done

### Convergence End-to-End
- Added `RecordConsensusPosition(ctx, checkpointID, position, explanation)` service method
  - Fetches checkpoint, verifies workspace membership, creates ConsensusSignal, persists, publishes event
- Added `GetCheckpoint(ctx, checkpointID)` service method
- Added `FindCheckpointsByWorkspace(ctx, wsID, openOnly)` with `?status=open` query param support
- Added 3 new HTTP endpoints:
  - `POST /api/v1/checkpoints/{checkpointId}/signals` — record align/concern/block
  - `GET /api/v1/checkpoints/{checkpointId}` — get checkpoint with signals
  - `GET /api/v1/workspaces/{wsId}/checkpoints` — list checkpoints
- Added `RecordConsensusPositionRequest` DTO
- 16 service tests covering: happy path, auth, membership, invalid position, duplicates, resolved checkpoint, consensus promotion, no-consensus no-promotion

### Deliverable Finalization
- Added `finalized` bool field to Deliverable domain entity
- Added `Finalize()` domain method — sets flag, rejects double finalize
- `UpdateContent()`, `ChangeFormat()`, `AddSourceLeaf()` all reject edits when finalized
- Updated `ReconstructDeliverable` signature with `finalized bool` param
- Migration: `ALTER TABLE deliverables ADD COLUMN finalized BOOLEAN NOT NULL DEFAULT FALSE` (applied)
- Updated sqlc queries, regenerated, mapper updated with `finalized` column
- Updated `DeliverableResponse` DTO with `finalized` field
- Service `Finalize()` now calls domain `Finalize()` before persisting
- 5 new domain tests: sets flag, rejects double, rejects edit/format/source after finalize

### Notification: In-App Transport + MarkRead
- Created `internal/notification/adapter/inapp/transport.go` — implements `NotificationTransport`
  - No-op `Send()`: for in-app, notification is already persisted and queryable via API
  - Service handles `MarkDelivered` transition on transport success
- Wired in composition root: replaced `nil` with `[]NotificationTransport{inapp.NewTransport()}`
- Added `FindByID` to `NotificationRepository` port + adapter + sqlc query
- Implemented `MarkRead` service method: fetch by ID, ownership check, domain transition, persist
- Updated test mock with `FindByID` stub

## Decisions made
- In-app transport is a no-op adapter — the notification is "delivered" the moment it's in the database and queryable. The transport interface exists so push/email can be added later with real external calls.
- `MarkRead` enforces ownership: `notif.UserID() != callerID` → unauthorized
- Deliverable `finalized` is a boolean, not a status enum — deliverables don't have a lifecycle complex enough to warrant a state machine

## Open questions
- None

## Next steps
- Synthesis live verification (already implemented, needs smoke test with LLM)
- Priority 1 frontend blockers (graph queries, batch position, pagination, search)
- Or: start frontend (SvelteKit)
