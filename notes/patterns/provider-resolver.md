# Provider Resolver (Per-Context Resource Resolution)

## What
A resolver wraps a port interface with context-dependent lookup. Instead of injecting a single concrete dependency, inject a resolver that picks the right implementation at runtime based on a context key (e.g., workspace ID).

## Why
When a shared resource (like an LLM provider) can be configured differently per tenant/workspace/user, the consuming service shouldn't know about the lookup logic. The resolver absorbs that complexity: check cache → load config from DB → create instance → cache → return. Consumers just call `Resolve(ctx, key)` and get back the same interface they'd use directly.

## Example
```go
// Interface (pkg/llm/)
type ProviderResolver interface {
    Resolve(ctx context.Context, workspaceID types.WorkspaceID) (Provider, error)
}

// Consumer (session service) — resolves per call
provider, err := s.llm.Resolve(ctx, session.WorkspaceID())
resp, err := provider.ChatCompletion(ctx, req)

// Adapter (composition root) — implements the resolution
type llmProviderResolver struct {
    llmConfigs  LLMConfigRepository  // read workspace config
    cipher      Encryptor            // decrypt API key
    fallback    Provider             // server-wide default
    cache       map[string]Provider  // keyed by workspace ID
}
```

## Gotchas
- Cache invalidation is critical — when config changes, stale providers serve wrong credentials. Use event subscribers to call `InvalidateCache`.
- The resolver lives in the composition root (`cmd/server/adapters.go`), not in the port package — it's a wiring concern.
- Fallback must always be available — if workspace has no config, return the server-wide default, never error.
- Thread safety: the cache needs `sync.RWMutex` since multiple goroutines resolve concurrently.
- Don't over-cache: if providers hold connections (e.g., Gemini), cached entries may go stale over time. Consider TTL eviction for production.

## Related
[[consumer-side-ports]], [[provider-wiring]], [[fire-and-forget-events]]
