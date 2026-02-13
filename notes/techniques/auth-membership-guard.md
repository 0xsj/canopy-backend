# Auth and Membership Guards

## What

Service methods extract JWT claims from `context.Context`, resolve the caller's identity, and validate membership (org or workspace) through injected read ports before performing any domain operation.

## Why

- Auth enforcement happens at the service layer, not in handlers or domain
- Membership checks use the same injected interfaces as other cross-context reads
- Consistent pattern across all services — easy to audit for missing auth checks
- Returns typed sentinel errors (`ErrUnauthenticated`, `ErrUnauthorized`) that HTTP handlers can map to status codes

## Example

### Basic Membership Guard

```go
func (s *Service) requireMember(ctx context.Context, workspaceID types.WorkspaceID) (types.UserID, error) {
    claims, ok := auth.FromClaims(ctx)
    if !ok {
        return types.UserID{}, canopyerr.ErrUnauthenticated
    }
    callerID := types.UserIDFrom(claims.Subject)

    if _, err := s.wsMembers.FindMember(ctx, workspaceID, callerID); err != nil {
        if canopyerr.GetKind(err) == canopyerr.KindNotFound {
            return types.UserID{}, canopyerr.ErrUnauthorized
        }
        return types.UserID{}, err
    }
    return callerID, nil
}
```

### Role-Based Guard (Organization)

```go
func (s *Service) requireRole(ctx context.Context, orgID types.OrgID, minimum domain.Role) (types.UserID, error) {
    claims, ok := auth.FromClaims(ctx)
    if !ok {
        return types.UserID{}, canopyerr.ErrUnauthenticated
    }
    callerID := types.UserIDFrom(claims.Subject)

    member, err := s.members.FindMember(ctx, orgID, callerID)
    if err != nil {
        if canopyerr.GetKind(err) == canopyerr.KindNotFound {
            return types.UserID{}, canopyerr.ErrUnauthorized
        }
        return types.UserID{}, err
    }

    if !hasMinimumRole(member.Role(), minimum) {
        return types.UserID{}, canopyerr.ErrUnauthorized
    }
    return callerID, nil
}

var roleHierarchy = map[domain.Role]int{
    domain.RoleMember: 0,
    domain.RoleAdmin:  1,
    domain.RoleOwner:  2,
}

func hasMinimumRole(actual, minimum domain.Role) bool {
    return roleHierarchy[actual] >= roleHierarchy[minimum]
}
```

## Usage in Service Methods

```go
func (s *Service) Plant(ctx context.Context, workspaceID types.WorkspaceID, ...) (domain.Seed, error) {
    const op = "seed: plant"

    callerID, err := s.requireMember(ctx, workspaceID)
    if err != nil {
        return domain.Seed{}, canopyerr.Wrap(err, op)
    }

    // callerID is now trusted — use it as the author
    seed, err := domain.NewSeed(workspaceID, callerID, title, description)
    // ...
}
```

## Guard Variants Across Contexts

| Guard | Used By | Checks |
|---|---|---|
| `requireMember(ctx, workspaceID)` | seed, exploration, discussion, convergence, session, synthesis, deliverable | Workspace membership |
| `requireRole(ctx, orgID, minimum)` | organization | Org membership + role hierarchy |
| `requireLoreKeeper(ctx, workspaceID)` | workspace | Workspace membership + LoreKeeper role |
| `authenticatedUserID(ctx)` | notification | Just auth, no membership (user-scoped) |

## Gotchas

- `claims.Subject` is the **external ID** from the OAuth provider, mapped to a `UserID` via `types.UserIDFrom()` — not the raw string
- `KindNotFound` from the membership lookup means "not a member" → return `ErrUnauthorized`, not `ErrNotFound`
- The identity context has **no** membership guard — user operations are global (register, find self)
- The ledger context has **no** auth checks — it's an internal sink called by event subscribers, not user-facing

## Related

- [[consumer-side-ports]] — `WorkspaceMemberReader` is the read port that makes membership checks possible
- [[error-chain-metadata]] — `ErrUnauthenticated` and `ErrUnauthorized` carry kind metadata for HTTP status mapping
- [[context-propagation-logger]] — claims and logger both travel in context, set by middleware
