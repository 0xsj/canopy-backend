# 2026-02-14 — Event Subscribers: Notification + Audit Expansion

## What was done

### Notification subscriber
- Built `cmd/server/notification_subscriber.go` — registry-driven notification handler
- 11 event types mapped to in-app notifications:
  - `workspace.member.joined`, `workspace.phase.transitioned`
  - `exploration.leaf.created`, `exploration.branch.created`
  - `discussion.comment.added`
  - `convergence.consensus.reached`, `convergence.leaf.promoted`
  - `session.completed`
  - `synthesis.completed`
  - `deliverable.finalized`
  - `seed.planted`
- Handler resolves workspace members via `FindByWorkspace`, excludes the actor (author/initiator), sends `in_app` notification per remaining member
- Registered as durable consumer `notification_events` in JetStream
- Always acks — partial delivery is acceptable, member lookup failures are logged and skipped

### Domain audit registry expansion
- Expanded `domainAuditRegistry` from 9 → 23 event types
- Added: seed constraints, workspace config/phase, connections, discussions, session started, convergence checkpoints/consensus, synthesis start/complete, deliverable drafts/updates

### Tests
- 8 unit tests for the notification handler:
  - Mapped event notifies members, excludes actor
  - No actor field notifies all members
  - Unmapped event silently acks
  - Empty workspace ID silently acks
  - Member lookup failure gracefully acks
  - Bad JSON gracefully acks
  - Registry entries have required fields
  - `notificationMappedEvents()` returns correct count

### Wiring
- `registerSubscribers` now accepts `notifService` + `workspaceMemberLister` deps
- `main.go` passes `notifP.Service` + `wsP.MemberRepo` to subscriber registration

## Files created
- `cmd/server/notification_subscriber.go`
- `cmd/server/notification_subscriber_test.go`

## Files modified
- `cmd/server/subscribers.go` — new consumer + expanded audit registry
- `cmd/server/main.go` — pass notification deps

## Decisions made
- Notification subscriber always acks (returns nil) — even on member lookup failure or partial send failure. Rationale: notifications are best-effort; a stuck subscriber blocking the JetStream queue is worse than a missed notification.
- Actor exclusion uses string comparison on user ID from the event payload. No auth context in subscriber handlers — they run outside HTTP request scope.
- All notifications use `ChannelInApp` for now. When push/email transports are wired, the handler can be extended to check user subscription preferences.
- `workspaceMemberLister` interface defined in the composition root — follows consumer-side port pattern. Satisfied by `wsP.MemberRepo` directly.

## Open questions
- Should notification subscriber respect user subscription preferences (digest frequency, channel selection) per workspace?
- Should we add rate limiting to prevent notification spam during bulk operations (e.g., synthesis creating many leaves)?

## Next steps
- Remaining handler smoke tests
- LLM adapter for sessions/synthesis
