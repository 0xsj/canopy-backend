# 2026-02-11 — Config Package

## What was done
- Built `pkg/config` — config toolkit for env-based configuration
- Implemented: EnvReader with prefix nesting and type conversion, Section interface, Loader for composing and validating sections, validation helpers (Errors collector)
- Updated `cmd/demo/main.go` to wire config → logger → errors showing all three packages together
- Demo includes ServerConfig and DatabaseConfig as example sections

## Decisions made
- Config is a toolkit, not a monolithic struct — each bounded context declares its own Section
- Environment variables only, no config files, no viper — stdlib is sufficient
- EnvReader with prefix nesting: `CANOPY_SERVER_PORT` from prefix "CANOPY" + section "SERVER" + key "PORT"
- Sections implement Load(EnvReader) + Validate() — the Loader orchestrates both
- Validation uses an Errors collector pattern — accumulates all failures before reporting
- Sensible dev defaults so `go run` works without setting env vars
- When a context becomes its own microservice, it just changes its root prefix

## Open questions
- None for config package

## Next steps
- Build shared types package or move into domain types
