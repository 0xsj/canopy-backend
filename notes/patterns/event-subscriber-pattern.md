# Event Subscriber Pattern

## What

Durable JetStream consumers subscribe to event subjects and call service methods to react to cross-context events. Subscribers are registered at startup in the composition root and cleaned up on shutdown. Two patterns exist: **audit handlers** (log everything) and **notification handlers** (fan-out to workspace members).

## Why

- Cross-context reactions without direct method calls — contexts stay decoupled
- Durable consumers survive restarts — missed events are redelivered
- Ack/nak semantics: return `nil` to ack, return `error` to trigger redelivery
- Central registration in `cmd/server/subscribers.go` makes it easy to see all wired reactions
- Registry-driven: new event types are added by extending a map, not writing new handlers

## Example

### Registration (Multiple Consumers)

```go
func registerSubscribers(
    ctx context.Context,
    sub events.Subscriber,
    ledger *ledgerservice.Service,
    notif *notifservice.Service,
    members workspaceMemberLister,
    log logger.Logger,
) (func(), error) {
    var subs []events.Subscription

    systemSub, _ := sub.Subscribe(ctx, "workspace.>",
        systemAuditHandler(ledger, log),
        events.WithConsumer("ledger_system_audit"))
    subs = append(subs, systemSub)

    notifSub, _ := sub.Subscribe(ctx, "workspace.>",
        notificationHandler(notif, members, log),
        events.WithConsumer("notification_events"))
    subs = append(subs, notifSub)

    return func() { cleanupAll(subs) }, nil
}
```

### Fan-Out Notification Handler

```go
func notificationHandler(notif *notifservice.Service, members workspaceMemberLister, log logger.Logger) events.Handler {
    return func(ctx context.Context, event events.Event) error {
        mapping, ok := notificationRegistry[event.Type]
        if !ok {
            return nil // ack — not a mapped event
        }
        // Parse event → find workspace members → exclude actor → send per-member
        wsMembers, _ := members.FindByWorkspace(ctx, wsID)
        for _, m := range wsMembers {
            if m.UserID().String() == actorID { continue }
            notif.Send(ctx, m.UserID(), ChannelInApp, mapping.title, ...)
        }
        return nil // always ack — partial delivery is acceptable
    }
}
```

### Registry-Driven Mapping

```go
var notificationRegistry = map[string]notificationMapping{
    "exploration.leaf.created": {
        title: "New leaf created", resourceType: "leaf",
        resourceField: "leaf_id", actorField: "author_id",
    },
    "convergence.consensus.reached": {
        title: "Consensus reached", resourceType: "checkpoint",
        resourceField: "checkpoint_id", actorField: "", // notify all
    },
}
```

## Current Subscribers

| Consumer | Subject | Durable | Purpose |
|---|---|---|---|
| `ledger_system_audit` | `workspace.>` | Yes | Raw JSON audit of every event |
| `ledger_domain_audit` | `workspace.>` | Yes | Semantic audit of 23 mapped events |
| `notification_events` | `workspace.>` | Yes | In-app notifications for 11 event types |
| WebSocket bridge | `workspace.>` | No | Broadcast to connected clients |

## Gotchas

- Return `nil` from handler to ack even on parse errors — bad data won't self-resolve on redelivery
- Return `error` only for transient failures (DB down) — JetStream will redeliver
- The handler closure captures the service — no global state
- `events.WithConsumer("name")` creates a durable consumer; omit for ephemeral
- JetStream streams can't have overlapping subjects — all subscribers share the `canopy` stream
- Notification handlers run outside HTTP request scope — no auth context available. Actor ID comes from the event payload, not `auth.FromClaims(ctx)`
- Fan-out always acks: a stuck notification subscriber blocking the queue is worse than a missed notification

## Related

- [[fire-and-forget-events]] — the publishing side of this pattern
- [[consumer-side-ports]] — synchronous alternative for cross-context reads
