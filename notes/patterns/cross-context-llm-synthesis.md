# Cross-Context LLM Synthesis Pattern

## What
How to implement an LLM-powered workflow that reads from multiple bounded contexts, calls an LLM, and writes the result back to another context — all while maintaining hexagonal architecture boundaries.

## Why
Synthesis workflows need data from seed context (topic info), exploration context (source leaves), and write results back to exploration context (new synthesis leaf). Direct imports between contexts would violate bounded context isolation.

## Example
The synthesis service defines consumer-side ports:
- `SourceLeafReader` — reads leaves (maps exploration domain → synthesis value types)
- `SeedReader` — reads seeds (maps seed domain → synthesis value types)
- `SynthesisLeafCreator` — creates branch + leaf atomically (adapter uses db.WithTx)

Adapters live in `cmd/server/adapters.go` (the composition root). They translate between domain types, keeping each context independent.

The service flow: auth → create workflow (pending→processing) → load source data → build prompt → call LLM → parse response → create leaf → complete workflow.

If any step after workflow creation fails, the workflow is marked "failed" with a reason (graceful degradation, no error returned).

## Gotchas
- LLM JSON output parsing needs to handle markdown fences (LLMs often wrap JSON in ```json blocks)
- Synthesis leaf creation needs a branch — the adapter creates one atomically via transaction
- Seed ID is derived from the first source leaf; source leaves could theoretically span multiple seeds
- The tx factory pattern (func(DBTX) Repository) is needed for the leaf creator adapter to create repos within a transaction scope

## Related
[[provider-wiring]], [[transaction-factory-injection]], [[consumer-side-ports]], [[fire-and-forget-events]]
