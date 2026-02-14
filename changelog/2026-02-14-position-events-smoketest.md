# 2026-02-14 — Canvas Position, Event Subscribers, Smoke Tests

## What was done

### X/Y Position for Seeds and Leaves
- Added `position_x`, `position_y` (FLOAT8) to seeds and leaves tables via ALTER TABLE migrations
- Updated domain entities: `positionX`, `positionY` fields, `UpdatePosition` methods, `ReconstructSeed`/`ReconstructLeaf` signatures
- Updated repository ports: `UpdatePosition(ctx, id, x, y float64) error` on both `SeedRepository` and `LeafRepository`
- Regenerated sqlc for both seed and exploration adapters
- Updated mappers (create params + domain reconstruction)
- Updated repository implementations with `UpdatePosition` methods
- Added service methods: `seed.UpdatePosition`, `exploration.UpdateLeafPosition`
- Added HTTP endpoints: `PATCH /api/v1/seeds/{seedId}/position`, `PATCH /api/v1/leaves/{leafId}/position`
- Updated response DTOs to include `position_x`, `position_y`
- Fixed `context_assembler_test.go` (broken by ReconstructSeed/ReconstructLeaf signature changes)
- Branches intentionally excluded — they are grouping concepts, not canvas nodes

### Event Subscriber: LLM Config Cache Invalidation
- Added `llmConfigHandler` in `cmd/server/subscribers.go`
- Subscribes to `workspace.*.workspace.workspace.llm_config.*`
- Calls `resolver.InvalidateCache(workspaceID)` on config update/delete events
- Added LLM config events to domain audit registry (`workspace.llm_config.updated`, `workspace.llm_config.deleted`)
- Wired into `registerSubscribers()` — now 4 durable consumers total

### Migrations Applied
- `llm_configs` table (BYOM) — CREATE TABLE with workspace FK
- `seeds.position_x`, `seeds.position_y` — ALTER TABLE ADD COLUMN
- `leaves.position_x`, `leaves.position_y` — ALTER TABLE ADD COLUMN

### Smoke Tests
- All existing endpoints verified working
- Seed position: PATCH update + GET verify (150.5, 300.25)
- Leaf position: PATCH update + GET verify (250.0, 450.75)
- BYOM: SET config (openai/gpt-4o) → GET (masked key: `sk-...7890`) → UPDATE (anthropic) → DELETE → GET (404)
- `go build ./...` clean, `go test ./...` all pass

## Decisions made
- sqlc SELECT column order must match actual table column order (ALTER TABLE appends at end) to avoid generating separate Row types
- Leaf `UpdatePosition` does not touch timestamps — position is a presentation concern, not content mutation
- Seed `UpdatePosition` does touch `updated_at` (seeds are mutable, this is consistent with other seed updates)

## Open questions
- None

## Next steps
- Frontend (SvelteKit)
