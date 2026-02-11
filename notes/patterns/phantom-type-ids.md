# Phantom Type IDs

## What
A generic ID type parameterized by an unexported tag type that exists only at compile time. The tag prevents accidental mixing of IDs across entity boundaries — you can't pass a `LeafID` where a `UserID` is expected.

## Why
In a system with many entity types, raw `string` or `uuid.UUID` IDs are interchangeable at the type level. A function accepting `string` for both `userID` and `leafID` has no compile-time protection against swapping them. Phantom types make the distinction explicit.

## Example
```go
// Unexported tag types — exist only for the type parameter.
type userTag struct{}
type leafTag struct{}

// Generic ID with phantom type.
type ID[T any] struct {
    value string
}

// Type aliases for ergonomic use.
type UserID = ID[userTag]
type LeafID = ID[leafTag]

// This won't compile — different phantom types.
func getLeaf(id LeafID) { ... }
getLeaf(NewUserID()) // compile error
```

## Gotchas
- Tag types must be unexported — if exported, external packages can construct arbitrary IDs.
- Type aliases (`=`) are used instead of type definitions so the generic methods remain accessible without re-declaration.
- JSON marshaling works on the generic type — no need for per-entity marshal implementations.
- The phantom type has zero runtime cost — it's erased during compilation.

## Related
[[config-section-pattern]]
