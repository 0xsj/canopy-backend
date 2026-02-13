# Fire-and-Forget Event Publishing

## What

Domain events are published **after** successful persistence. Publish failures are logged but never returned to the caller. The publisher doesn't know or care who subscribes.

## Why

- The primary operation (persist to database) must not fail because of event delivery issues
- Events are the API between bounded contexts — they decouple producer from consumer
- Subscribers are eventually consistent by design; missing an event is recoverable
- Keeps the service method's error path clean — callers only see domain and persistence errors

## Example

Every service with events has a private `publish` helper:

```go
func (s *Service) publish(ctx context.Context, eventType, workspaceID string, data any) {
    event, err := events.New(eventType, workspaceID, data)
    if err != nil {
        s.log.Error("event creation failed",
            logger.String("type", eventType), logger.Err(err))
        return
    }
    event.Subject = events.BuildSubject(workspaceID, "organization", eventType)
    if pubErr := s.pub.Publish(ctx, event); pubErr != nil {
        s.log.Error("event publish failed",
            logger.String("type", eventType), logger.Err(pubErr))
    }
}
```

Called at the end of a successful operation:

```go
func (s *Service) CreateOrg(ctx context.Context, ...) (domain.Organization, error) {
    // 1. validate
    // 2. persist (if this fails, return error)
    // 3. publish (if this fails, log and continue)
    s.publish(ctx, domain.SubjectOrgCreated, org.ID().String(), domain.OrgCreatedData{...})
    return org, nil
}
```

## Subject Convention

```
workspace.{workspace_id}.{context}.{event_type}
```

Built with `events.BuildSubject(workspaceID, context, eventType)`.

Examples:
- `workspace.abc123.organization.org.created`
- `workspace.abc123.exploration.leaf.created`
- `workspace.abc123.convergence.consensus.reached`

## Event Payload Location

Event type constants and payload structs live in each context's `domain/events.go`, not in the shared `pkg/events/` package. This keeps contexts decoupled from a global event schema (see ADR-006).

## Gotchas

- The ledger context is a **sink** — it subscribes to events but publishes none. It has no `events.Publisher` dependency.
- Event data should carry only what subscribers need. Don't serialize the entire entity.
- `events.New()` marshals the data payload to `json.RawMessage` — if the data struct has broken JSON tags, the event creation itself fails (caught by the error check)
- Publishing happens **outside** any transaction — if you publish inside `WithTx`, a rollback won't un-publish the event

## Related

- [[consumer-side-ports]] — the other cross-context communication mechanism (synchronous reads)
- [[error-chain-metadata]] — publish errors are logged with structured fields, not wrapped
