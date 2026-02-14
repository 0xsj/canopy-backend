# 2026-02-14 — LLM Adapter: Provider Port + Three Adapters

## What was done

### pkg/llm — Provider port and adapters
- Defined `Provider` interface with single `ChatCompletion(ctx, ChatRequest) (ChatResponse, error)` method
- Request/response types: `ChatRequest`, `ChatResponse`, `Message`, `Options`, `Usage`
- Config section: `CANOPY_LLM_{PROVIDER,MODEL,API_KEY,REQUEST_TIMEOUT,TEMPERATURE,MAX_TOKENS}`
- Three provider adapters:
  - **Anthropic** (`anthropic.go`) — extracts system messages to separate `System` param, iterates content blocks for text extraction, maps `anthropic.Model` to string
  - **OpenAI** (`openai.go`) — uses `SystemMessage`/`UserMessage`/`AssistantMessage` helpers, reads `Choices[0].Message.Content`
  - **Gemini** (`gemini.go`) — sets `SystemInstruction` on model, uses `StartChat` + `SendMessage` for multi-turn, reads response candidates
- Factory (`factory.go`) — `NewProvider(ctx, cfg, log)` routes to correct adapter based on `cfg.Provider`
- Updated `doc.go` to reflect implemented state

### pkg/config — Float64 support
- Added `EnvReader.Float64(name, fallback)` method for temperature config

### Session service — LLM integration
- Added `llm.Provider` field to `Service` struct
- `AddMessage` now: persists user message → calls `assembler.Assemble()` → calls `llm.ChatCompletion()` → appends assistant response → persists again
- If LLM call fails, user message is still persisted; error is returned and logged

### Wiring
- `internal/session/provider.go` — `Wire()` accepts `llm.Provider` param
- `cmd/server/main.go` — loads LLM config, creates provider via factory, passes to `session.Wire()`

### Tests (19 passing)
- `config_test.go` — 9 tests: defaults, env overrides, validation (valid config, all 3 providers, missing/invalid fields)
- `factory_test.go` — 5 tests: Anthropic/OpenAI routing, unknown/empty provider errors, compile-time interface check for all 3 adapters
- `provider_test.go` — 5 tests: mock provider implements interface, canned response, request capture, error propagation

## Files created
- `pkg/llm/provider.go`
- `pkg/llm/config.go`
- `pkg/llm/anthropic.go`
- `pkg/llm/openai.go`
- `pkg/llm/gemini.go`
- `pkg/llm/factory.go`
- `pkg/llm/config_test.go`
- `pkg/llm/factory_test.go`
- `pkg/llm/provider_test.go`

## Files modified
- `pkg/config/env.go` — added `Float64()` method
- `pkg/llm/doc.go` — updated to reflect implementation
- `internal/session/service/service.go` — added `llm.Provider` field, modified `AddMessage`
- `internal/session/provider.go` — accepts `llm.Provider` in `Wire()`
- `cmd/server/main.go` — LLM config + provider wiring

## Dependencies added
- `github.com/anthropics/anthropic-sdk-go` v1.22.1
- `github.com/openai/openai-go` v1.12.0
- `github.com/google/generative-ai-go` v0.20.1
- `google.golang.org/api` v0.266.0 (transitive, for Gemini `option.WithAPIKey`)

## Decisions made
- Single `ChatCompletion` method on the Provider interface — `StructuredOutput` and `Synthesis` are prompt engineering on top of chat, not separate API calls. Can add later if needed.
- Single API key in config (not per-provider) — only one provider is active at a time for MVP. BYOK (per-request keys) is a future feature.
- No `Close()` on Provider interface — Anthropic and OpenAI clients are stateless HTTP. Gemini gRPC client cleanup happens at process exit.
- Gemini factory test skipped — `newGeminiProvider` makes a real gRPC connection. Would need `//go:build integration` tag.
- `noopContextAssembler` stays in place — real implementation needs seed/leaf reader ports (separate task).

## Open questions
- Should the ContextAssembler pull seed constraints and parent leaf content for exploration sessions?
- Should we add request timeout enforcement via `context.WithTimeout` in the adapters?

## Next steps
- ContextAssembler real implementation (needs seed/leaf reader cross-context ports)
- Remaining handler smoke tests
- Synthesis LLM integration
