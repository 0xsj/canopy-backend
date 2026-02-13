# Implicit Interface Satisfaction Check

## What

Go interfaces are satisfied implicitly — there's no `implements` keyword. A compile-time assertion using a blank identifier ensures a concrete type satisfies an interface:

```go
var _ domain.OrgRepository = (*OrgRepository)(nil)
```

## Why

- Go's implicit interface satisfaction means you won't know you're missing a method until something tries to use the type as that interface — which might be in a different package or at runtime
- The blank identifier assignment triggers a compile-time check at the declaration site
- Catches missing or mismatched method signatures immediately when you modify the interface or the type
- Documents which interface the type is intended to satisfy

## Example

```go
package postgres

import "github.com/0xsj/canopy-backend/internal/organization/domain"

type OrgRepository struct {
    q *sqlc.Queries
}

// Compile-time check: OrgRepository must implement domain.OrgRepository
var _ domain.OrgRepository = (*OrgRepository)(nil)

func (r *OrgRepository) Create(ctx context.Context, org domain.Organization) error { ... }
func (r *OrgRepository) FindByID(ctx context.Context, id types.OrgID) (domain.Organization, error) { ... }
// ... all methods from the interface
```

If you remove `FindByID`, the build fails at the `var _` line — not at the composition root where it's first used.

## How It Works

- `(*OrgRepository)(nil)` creates a nil pointer of type `*OrgRepository`
- Assigning it to `domain.OrgRepository` (interface type) forces the compiler to verify all methods exist
- The blank identifier `_` discards the value — no runtime allocation, no memory cost
- It's a pure compile-time artifact

## Convention

Place the assertion right after the type declaration, before any methods:

```go
type OrgRepository struct { ... }

var _ domain.OrgRepository = (*OrgRepository)(nil)

func NewOrgRepository(db database.DBTX) *OrgRepository { ... }
func (r *OrgRepository) Create(...) error { ... }
```

## Gotchas

- Only catches **method signature** mismatches, not behavioral correctness
- Uses pointer receiver check (`*OrgRepository`) — if the interface is satisfied by value receiver methods only, `OrgRepository` (non-pointer) would also work, but pointer is the safer default
- Some linters (like `golangci-lint` with `interfacer`) can detect missing implementations, but the blank assignment is more explicit and doesn't require tooling

## Related

- [[consumer-side-ports]] — the interfaces being checked are often cross-context read ports
- [[sqlc-mapper-pattern]] — every adapter repo uses this pattern to verify it satisfies the domain port
