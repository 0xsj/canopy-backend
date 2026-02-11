# Discriminated Union API Response

## What
An API response envelope that is either a success (carrying data) or an error (carrying error info), never both. In Go, modeled as a generic struct with mutually exclusive fields and constructors that enforce the invariant.

## Why
The frontend receives a consistent shape regardless of which bounded context handled the request. Parsing is predictable: check for `error`, if absent read `data`. No ambiguous states where both or neither are present.

Three variants cover all cases:
- `OK[T]` — success with data
- `Fail[T]` — domain error (not found, conflict, etc.)
- `FailWithDetails[T]` — validation error with field-level feedback

## Example
```go
// Constructors enforce the union.
resp := types.OK(leaf)                    // {"data": {...}}
resp := types.Fail[Leaf]("not_found", "leaf not found")  // {"error": {...}}
resp := types.FailWithDetails[Leaf](      // {"error": {..., "details": [...]}}
    "validation_failed", "invalid",
    []types.FieldError{{Field: "title", Message: "is required"}},
)

// Validate catches bad states.
bad := types.Response[Leaf]{Data: &leaf, Error: &apiErr}
bad.Validate() // error: both data and error are set
```

## Gotchas
- Go doesn't have real sum types — the invariant is enforced by constructors and checked by `Validate()`, but the struct fields are exported for JSON marshaling.
- `omitempty` on pointer fields ensures clean JSON — `data` or `error`, never both keys present.
- `FieldError` details are for client-side form mapping. They don't leak internal error chains — those stay in server logs.

## Related
[[builder-pattern-errors]]
