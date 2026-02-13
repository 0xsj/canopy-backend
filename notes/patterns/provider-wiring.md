# Provider Pattern (Context Wiring)

## What

Each bounded context exposes a `Wire()` function in `internal/{context}/provider.go` that encapsulates the full repo → service → handler dependency chain. The composition root calls `Wire()` once per context and gets back a `*Provider` struct with public access to the handler and any cross-context dependencies.

## Why

- Eliminates ~6 constructor calls per context from `main.go` — replaced by one `Wire()` call
- Import list in the composition root drops from ~55 entries to ~20
- Each context owns its internal wiring — adding a repo or changing a constructor signature only touches `provider.go`
- Cross-context dependencies are explicit: only repos needed by other contexts are exposed on the Provider struct

## Example

```go
// internal/seed/provider.go
package seed

type Provider struct {
    Service *service.Service
    Handler *v1.Handler
}

func Wire(db *database.DB, wsMemberRepo domain.WorkspaceMemberReader, pub events.Publisher, log logger.Logger) *Provider {
    q := sqlc.New(db.Pool())
    seedRepo := postgres.NewSeedRepository(q)

    svc := service.New(seedRepo, wsMemberRepo, pub, log)
    handler := v1.NewHandler(svc, log)

    return &Provider{Service: svc, Handler: handler}
}
```

At composition root:

```go
seedP := seed.Wire(db, wsP.MemberRepo, pub, log)
seedP.Handler.Register(apiMux)
```

## Provider Struct Fields

| Context | Exposes | Why |
|---|---|---|
| identity | Service, Handler, **UserRepo** | org needs `UserReader` |
| organization | Service, Handler, **MemberRepo** | workspace needs `OrgMemberReader` |
| workspace | Service, Handler, **MemberRepo** | 7 contexts need `WorkspaceMemberReader` |
| exploration | Service, Handler, **LeafRepo** | convergence needs `LeafWriter` |
| seed | Service, Handler | No cross-context consumers |
| discussion | Service, Handler | No cross-context consumers |
| convergence | Service, Handler | No cross-context consumers |
| session | Service, Handler | No cross-context consumers |
| synthesis | Service, Handler | No cross-context consumers |
| deliverable | Service, Handler | No cross-context consumers |
| notification | Service, Handler | No cross-context consumers |
| ledger | Service, Handler | No cross-context consumers |

## Gotchas

- Transaction factories are created inside `Wire()` — they close over `sqlc.New()` and return a new repo per tx
- Cross-context adapters (e.g., `userReaderAdapter`) remain in `cmd/server/adapters.go`, not in provider files — they bridge between contexts and belong to the composition root
- The `Wire()` signature varies per context based on its cross-context port needs — no shared interface

## Related

- [[transaction-factory-injection]] — factories are set up inside Wire()
- [[consumer-side-ports]] — cross-context read ports are passed as Wire() parameters
- [[fire-and-forget-events]] — Publisher is passed to Wire() for event publishing
