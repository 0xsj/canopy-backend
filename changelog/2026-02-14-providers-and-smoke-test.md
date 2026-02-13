# 2026-02-14 — Providers, Composition Root, and Smoke Test

## What was done
- Created 12 provider files (`internal/{context}/provider.go`) that encapsulate repo → service → handler wiring
- Rewrote `cmd/server/main.go` sections 7-10: replaced ~90 lines (46 constructor calls) with ~14 Wire() calls
- Import block reduced from 55 entries to 20
- Ran all 12 database migrations against live Postgres
- Smoke tested the core flow: register user → create org → create workspace → plant seed → start branch → create leaf
- Fixed route conflict: `GET /api/v1/orgs/by-slug/{slug}` → `GET /api/v1/orgs?slug=xxx` (Go 1.22+ ServeMux ambiguity)
- Fixed FK violation in `StartBranch`: reordered transaction to insert branch (NULL root) → insert leaf → update branch with root_leaf_id

## Decisions made
- Provider pattern: each context gets a `Wire()` function returning `*Provider` with `Service`, `Handler`, and optionally exposed repos
- Only 4 contexts expose repos (identity, organization, workspace, exploration) — only those needed for cross-context ports
- Cross-context adapters remain in `cmd/server/adapters.go` — they bridge between contexts and belong to the composition root
- Org slug lookup changed from path param to query param to avoid ServeMux conflict

## Open questions
- 7 handler groups untested at HTTP level (discussion, convergence, session, synthesis, deliverable, notification, ledger)
- No event subscribers wired — events fire into NATS but no consumers react
- `noopContextAssembler` still stubbed (session LLM integration)
- Notification transports still nil

## Next steps
- Wire event subscribers (cross-context reactions: ledger entries, notifications)
- Smoke test remaining 7 handler groups
- Build LLM adapter for session context
- Add at least one notification transport (in-app)
