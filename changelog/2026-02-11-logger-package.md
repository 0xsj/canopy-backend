# 2026-02-11 — Logger Package

## What was done
- Built `pkg/observability/logger` — structured logging port and implementations
- Implemented: level, field (typed constructors), Logger interface, context helpers, functional options, ANSI color utilities, noop implementation, console implementation
- Added `Origin()` and `OriginFrame()` to `pkg/errors` — walks error chain to find creation site
- Added `Frame.Short()` to `pkg/errors` — compact `file:line` format for log fields
- Updated `cmd/demo/main.go` showing errors + logger working together with context propagation
- Full test coverage for logger package (level, field, logger context, options, noop, console, color)
- Extended error tests for Origin, OriginFrame, Frame.Short

## Decisions made
- Logger grouped under `pkg/observability/logger/` with room for future `metrics/` and `tracing/` siblings
- Logger interface is 5 methods: Debug, Info, Warn, Error, With — kept minimal per CLAUDE.md
- Field type with typed constructors (String, Int, Err, etc.) instead of `keysAndValues ...any` — maps cleanly to zap/slog when swapping
- Context propagation via `WithContext`/`FromContext` helpers, not on the interface itself
- `FromContext` returns noop when no logger found — never nil, always safe to call
- Console logger is dev-mode default: colorized, timestamps, structured `key=value` fields
- Functional options pattern for console configuration
- `Origin()` walks to deepest canopy error (not outermost like `errors.As`)

## Open questions
- None for logger package

## Next steps
- Build `pkg/config` or `pkg/database` next in the shared layer
