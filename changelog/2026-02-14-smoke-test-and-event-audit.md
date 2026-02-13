# 2026-02-14 — Smoke Test Script and Event Subscriber Audit

## What was done
- Created `scripts/smoke-test.sh` — comprehensive bash smoke test covering all 12 bounded contexts (~67 HTTP calls)
- Script exercises the full lifecycle: identity → org → workspace → seed → exploration → discussion → convergence → session → synthesis → deliverable → notification → ledger
- Audited the complete event landscape: 34 events defined, 31 actively published, 3 defined but not yet published
- Mapped all existing subscribers: ledger system audit, ledger domain audit, WebSocket bridge
- Identified all missing subscriber flows: notification (highest impact), convergence, synthesis, workspace
- Added two notes: provider-wiring pattern, event-subscriber pattern
- Created gitignored TODO.md for next session planning

## Decisions made
- Smoke test uses `call_accept_any` for endpoints where idempotency or missing state may produce alternative valid HTTP codes
- Fake user with `usr_` prefix used for member add/remove operations (no FK to identity)
- Notification mark-read with fabricated `ntf_` ID expects 404 (no subscriber wired to create notifications)
- TODO.md gitignored — session-specific planning artifact, not project documentation

## Open questions
- Notification subscriber: central (like ledger) vs per-context? Durable vs ephemeral?
- Which events should generate notifications, and for whom?
- Claims.Subject / user ID prefix mismatch in dev mode — needs resolution before full smoke test pass
- ADRs 002-007 referenced in memory but only 001 exists on disk — lost or never written?
- Architecture docs (vision.md, product.md, architecture.md, diagrams.md) referenced in CLAUDE.md but don't exist

## Next steps
- Run smoke test against live server, fix failures
- Design and implement notification subscriber
- Build LLM adapter for session context
- Write missing ADRs and architecture docs
