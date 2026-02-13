# Dual Constructor Pattern (New vs Reconstruct)

## What

Every domain entity has two constructors:

- **`New*()`** — for creation. Validates input, generates a fresh ID, sets initial timestamps. Returns `(Entity, error)`.
- **`Reconstruct*()`** — for database reads. Accepts all fields including ID and timestamps. No validation. Returns `Entity` (no error).

## Why

- Creation must enforce invariants (non-empty name, valid enum, etc.)
- Reconstruction from trusted persistence should never fail — the data was validated on write
- Keeps entity fields unexported — external code can't construct invalid state
- The separation makes it obvious which code path is "creating new" vs "loading existing"

## Example

```go
// Creation — validates and generates ID
func NewOrganization(name, slug string, ownerID types.UserID) (Organization, error) {
    if name == "" {
        return Organization{}, fmt.Errorf("organization: name is required")
    }
    if slug == "" {
        return Organization{}, fmt.Errorf("organization: slug is required")
    }
    if ownerID.IsZero() {
        return Organization{}, fmt.Errorf("organization: owner ID is required")
    }

    return Organization{
        id:         types.NewOrgID(),
        name:       name,
        slug:       slug,
        ownerID:    ownerID,
        settings:   make(map[string]any),
        timestamps: types.NewMutableTimestamps(),
    }, nil
}

// Reconstruction — trusts persisted data
func ReconstructOrganization(
    id types.OrgID,
    name string,
    slug string,
    personal bool,
    ownerID types.UserID,
    settings map[string]any,
    timestamps types.Timestamps,
) Organization {
    if settings == nil {
        settings = make(map[string]any)
    }
    return Organization{
        id: id, name: name, slug: slug,
        personal: personal, ownerID: ownerID,
        settings: settings, timestamps: timestamps,
    }
}
```

## Where Each Is Called

| Constructor | Called By | Context |
|---|---|---|
| `New*()` | Service layer | User action creating a new entity |
| `Reconstruct*()` | Adapter mapper (`*ToDomain`) | Loading from database |

## Gotchas

- `Reconstruct*()` should still handle nil maps/slices defensively (initialize to empty) to prevent nil pointer panics downstream
- Never call `Reconstruct*()` from service code — it bypasses invariants
- Never call `New*()` from adapter code — it would generate a new ID and overwrite the persisted one
- Immutable entities (Leaf, Signal, SystemEntry) use `NewTimestamps()` in `New*()`; mutable entities use `NewMutableTimestamps()`

## Related

- [[domain-state-machines]] — Reconstruct can set any state; New always starts at the initial state
- [[sqlc-mapper-pattern]] — mapper's `*ToDomain()` calls Reconstruct
- [[phantom-type-ids]] — New generates `types.NewXxxID()`; Reconstruct accepts `types.XxxIDFrom(string)`
