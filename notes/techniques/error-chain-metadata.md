# Error Chain Metadata Collection

## What
Walking a wrapped error chain to collect metadata from every layer, merging it into a single map. Outer (more recent) values take precedence over inner (origin) values.

## Why
In a layered architecture, each layer adds its own context as metadata when wrapping an error. The domain layer adds `leaf_id`, the service layer adds `workspace_id`, the handler doesn't add metadata but needs all of it for logging. Walking the chain collects everything in one pass.

## Example
```go
// Domain
errors.New("not found").WithMetadata("leaf_id", id)

// Service wraps and adds its context
errors.Wrap(err, "exploration: get leaf").WithMetadata("workspace_id", wsID)

// Handler collects all metadata for structured logging
meta := errors.CollectMetadata(err)
// meta = {"leaf_id": "abc", "workspace_id": "xyz"}
```

Implementation walks inner-first so outer values overwrite:
```go
func CollectMetadata(err error) Metadata {
    var chain []Metadata
    // collect from outer to inner...
    // apply from inner to outer (so outer wins)
    for i := len(chain) - 1; i >= 0; i-- {
        maps.Copy(collected, chain[i])
    }
}
```

## Gotchas
- The chain walk stops at the first non-canopy error. If a standard `fmt.Errorf` is in the middle of the chain, metadata below it is lost.
- `Wrap` gives the wrapper its own metadata map — it does not copy the cause's metadata. This is intentional: each layer owns its own context.

## Related
[[builder-pattern-errors]]
