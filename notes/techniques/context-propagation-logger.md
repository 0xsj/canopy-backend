# Context Propagation for Logger

## What
Storing a pre-configured logger in `context.Context` so every layer in the call chain can extract it without explicit parameter passing. Middleware creates the logger with request-scoped fields (request ID, user ID), stores it in context, and all downstream code pulls it out.

## Why
Request-scoped data like correlation IDs needs to appear on every log line. Passing the logger explicitly through every function signature is noisy. Context propagation keeps function signatures clean while ensuring structured fields are consistent across the entire request lifecycle.

## Example
```go
// Middleware creates a request-scoped logger.
requestLog := log.With(logger.String("request_id", reqID))
ctx := logger.WithContext(ctx, requestLog)

// Any downstream function extracts it.
func getLeaf(ctx context.Context, id string) error {
    log := logger.FromContext(ctx)
    log.Info("looking up leaf", logger.String("leaf_id", id))
    // Log line includes request_id automatically.
}
```

`FromContext` returns a noop logger when none is found — never nil, always safe.

## Gotchas
- The context key is an unexported struct type (`contextKey{}`) — prevents collisions with other packages.
- `FromContext` returning noop (not nil) means callers never need nil-checks. But it also means a missing logger is silent — no panic to tell you wiring is wrong. Use startup validation.
- `With()` creates a new logger instance — the parent is not mutated. Safe for concurrent use across goroutines handling different requests.

## Related
[[functional-options]]
