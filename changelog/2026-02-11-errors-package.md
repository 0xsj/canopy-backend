# 2026-02-11 — Errors Package

## What was done
- Built `pkg/errors` — the foundational error handling package
- Implemented: severity, kind, code, stack trace capture, base error interface, builder pattern, sentinels, wrap/unwrap utilities, chain metadata collection
- Full test coverage across all files (severity, kind, code, stack, error, sentinel, wrap)
- Created `cmd/demo/main.go` showing error propagation through domain → service → handler layers
- Added Makefile with fmt, test, test-verbose, test-coverage targets
- Added .gitignore for Go binaries and build artifacts

## Decisions made
- Errors carry classification (kind, code, severity) so handlers can translate to HTTP responses without type switches
- Builder pattern (`With*` methods) for error construction — earned by the number of attributes
- `Wrap` inherits kind/code/severity from the cause automatically — layers don't re-classify
- Stack traces captured at error creation site, not at each wrap point
- `CollectMetadata` walks the full chain, outer values take precedence over inner
- Request/correlation ID is a context + logger concern, not an errors concern
- Sentinel errors are package-level vars, bounded contexts wrap them with operation context

## Open questions
- None for errors package

## Next steps
- Build `pkg/logger` (structured logging with context-based request ID support)
- Build `pkg/config` or `pkg/database` next in the shared layer
