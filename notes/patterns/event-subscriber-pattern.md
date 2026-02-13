# Event Subscriber Pattern (Ledger Example)

## What

Durable JetStream consumers subscribe to event subjects and call service methods to react to cross-context events. Subscribers are registered at startup in the composition root and cleaned up on shutdown.

## Why

- Cross-context reactions without direct method calls — contexts stay decoupled
- Durable consumers survive restarts — missed events are redelivered
- Ack/nak semantics: return `nil` to ack, return `error` to trigger redelivery
- Central registration in `cmd/server/subscribers.go` makes it easy to see all wired reactions

## Example

### Registration

```go
func registerSubscribers(ctx context.Context, sub events.Subscriber, ledger *service.Service, log logger.Logger) (func(), error) {
    var subs []events.Subscription

    systemSub, err := sub.Subscribe(ctx, "workspace.>",
        systemAuditHandler(ledger, log),
        events.WithConsumer("ledger_system_audit"))
    if err != nil {
        return nil, err
    }
    subs = append(subs, systemSub)

    return func() { cleanupAll(subs) }, nil
}
```

### Handler

```go
func systemAuditHandler(ledger *service.Service, log logger.Logger) events.Handler {
    return func(ctx context.Context, event events.Event) error {
        // Parse, transform, persist
        if err := ledger.AppendSystem(ctx, event.Subject, data, sourceCtx); err != nil {
            return err // nak — triggers redelivery
        }
        return nil // ack
    }
}
```

### Domain Audit Registry (Data-Driven)

```go
var domainAuditRegistry = map[string]domainAuditMapping{
    "workspace.created": {action: ActionCreated, resourceType: "workspace", ...},
    "exploration.leaf.created": {action: ActionCreated, resourceType: "leaf", ...},
}
```

One handler iterates the registry — new event types are added by extending the map, not writing new handlers.

## Current Subscribers

| Consumer | Subject | Durable | Purpose |
|---|---|---|---|
| `ledger_system_audit` | `workspace.>` | Yes | Raw JSON audit of every event |
| `ledger_domain_audit` | `workspace.>` | Yes | Semantic audit of 10 mapped events |
| WebSocket bridge | `workspace.>` | No | Broadcast to connected clients |

## Gotchas

- Return `nil` from handler to ack even on parse errors — bad data won't self-resolve on redelivery
- Return `error` only for transient failures (DB down) — JetStream will redeliver
- The handler closure captures the service — no global state
- `events.WithConsumer("name")` creates a durable consumer; omit for ephemeral
- JetStream streams can't have overlapping subjects — all subscribers share the `canopy_events` stream

## Related

- [[fire-and-forget-events]] — the publishing side of this pattern
- [[consumer-side-ports]] — synchronous alternative for cross-context reads
