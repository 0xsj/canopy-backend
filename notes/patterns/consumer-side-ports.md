# Consumer-Side Ports (Cross-Context Read Interfaces)

## What

When a bounded context needs data from another context, it defines a **small read-only interface in its own package** — not in the provider's package. The composition root satisfies it with the provider's concrete adapter.

This is the hexagonal architecture rule: interfaces belong to the consumer, not the producer.

## Why

- Contexts never import each other's implementations — only domain types that appear in shared `pkg/types`
- The consumer declares exactly what it needs (1–3 methods), not the full repository
- Swapping providers or mocking for tests is trivial
- Adding a new consumer doesn't change the provider at all

## Example

```go
// In exploration/service/service.go — the consumer
type WorkspaceMemberReader interface {
    FindMember(ctx context.Context, workspaceID types.WorkspaceID, userID types.UserID) (wsdomain.WorkspaceMember, error)
}

type Service struct {
    repo      domain.LeafRepository
    wsMembers WorkspaceMemberReader  // cross-context port
    pub       events.Publisher
    log       logger.Logger
}
```

At composition root:

```go
// The workspace adapter satisfies the exploration service's port
wsRepo := wspostgres.NewWorkspaceMemberRepository(db)
explorationSvc := exploration.New(leafRepo, branchRepo, connRepo, wsRepo, db, ...)
```

## Current Port Map

| Port Interface | Defined In | Satisfied By |
|---|---|---|
| `UserReader` | organization/service | identity adapter |
| `OrgMemberReader` | workspace/service | organization adapter |
| `WorkspaceMemberReader` | seed, exploration, discussion, convergence, session, synthesis, deliverable | workspace adapter |
| `LeafWriter` | convergence/service | exploration adapter |

## Gotchas

- The port can reference another context's **domain types** (e.g., `wsdomain.WorkspaceMember`) for return values — this is acceptable because domain types are stable contracts
- Keep the port minimal. Don't copy the full repository interface — only the methods the consumer actually calls
- If the port grows beyond 3 methods, the consumer may be doing too much cross-context work — consider an event-based approach instead

## Related

- [[phantom-type-ids]] — IDs cross context boundaries safely via shared `pkg/types`
- [[fire-and-forget-events]] — the other cross-context communication channel
- [[config-section-pattern]] — same "consumer defines the contract" philosophy
