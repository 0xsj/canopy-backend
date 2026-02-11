# Builder Pattern for Errors

## What
Fluent `With*` methods on a concrete error struct that return `*self`, allowing chained construction. The concrete type is returned (not the interface) so builders are available at the call site, while the interface is used for consuming errors.

## Why
When an error carries many attributes (kind, code, severity, operation, metadata, cause), a constructor with positional args becomes unreadable. The builder lets callers set only what's relevant. It also reads well at the call site:

```go
errors.New("leaf not found").
    WithKind(errors.KindNotFound).
    WithCode("exploration_leaf_not_found").
    WithMetadata("leaf_id", id)
```

## Example
```go
// New returns *canopyError (concrete), not Error (interface).
// This makes builder methods available.
func New(message string) *canopyError { ... }

// Each builder returns the same pointer.
func (e *canopyError) WithKind(k Kind) *canopyError {
    e.kind = k
    return e
}
```

Consumers accept the interface:
```go
func handleError(err errors.Error) {
    kind := err.ErrorKind()
}
```

## Gotchas
- Return the concrete type from constructors, the interface from accessors. If `New` returned `Error`, builder methods wouldn't be visible.
- Builders mutate the receiver. Don't reuse a partially-built error across goroutines.

## Related
[[error-chain-metadata]]
