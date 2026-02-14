# 2026-02-14 — BYOM (Bring Your Own Model)

## What was done

### pkg/crypto/ — encryption package
- Created `pkg/crypto/crypto.go` — `Encryptor` interface + `AES256GCM` implementation
- Ciphertext format: nonce (12 bytes) || encrypted || tag (16 bytes)
- Created `pkg/crypto/config.go` — `CANOPY_CRYPTO_KEY` env var (hex-encoded 32-byte key)
- Follows standard `config.Section` pattern: `Load(EnvReader)` + `Validate() error`

### Workspace domain — LLMConfig entity
- Created `internal/workspace/domain/llm_config.go` — `LLMConfig` entity with `LLMProvider` value type
- Validation: provider must be anthropic/openai/gemini, model required, encrypted key required
- Standard dual-constructor pattern: `NewLLMConfig()` + `ReconstructLLMConfig()`
- Added `LLMConfigRepository` port to `repository.go` (Upsert/FindByWorkspace/Delete)
- Added `SubjectLLMConfigUpdated` / `SubjectLLMConfigDeleted` events to `events.go`

### Workspace Postgres adapter — migration + sqlc + repo
- Created migration `002_add_llm_configs.sql` — `llm_configs` table with FK cascade to workspaces
- Added 3 queries to `queries.sql`: UpsertLLMConfig (INSERT ON CONFLICT), FindLLMConfigByWorkspace, DeleteLLMConfig
- Regenerated sqlc models
- Created `llm_config_mapper.go` — domain ↔ sqlc mapping
- Created `llm_config_repository.go` — standard adapter pattern (const op, MapQueryError, RowsAffected)

### Workspace service — 3 new methods
- Added `Encryptor` interface (consumer-side port) to service package
- Added `llmConfigs` + `encryptor` fields to Service struct
- `SetLLMConfig` — encrypt key → upsert → publish event (requires lore keeper)
- `GetLLMConfig` — find → decrypt → return `LLMConfigResult` (requires member)
- `DeleteLLMConfig` — delete → publish event (requires lore keeper)

### Workspace HTTP handler — 3 new endpoints
- `PUT /api/v1/workspaces/{wsId}/llm-config` → handleSetLLMConfig
- `GET /api/v1/workspaces/{wsId}/llm-config` → handleGetLLMConfig (API key masked in response)
- `DELETE /api/v1/workspaces/{wsId}/llm-config` → handleDeleteLLMConfig
- Added `SetLLMConfigRequest` and `LLMConfigResponse` DTOs with `maskAPIKey()` helper

### pkg/llm/resolver.go — ProviderResolver interface
- `ProviderResolver.Resolve(ctx, workspaceID) (Provider, error)`
- Abstracts per-workspace provider resolution from consumer services

### Session + synthesis — Provider → ProviderResolver swap
- Both services now accept `llm.ProviderResolver` instead of `llm.Provider`
- `AddMessage` resolves provider by workspace ID before calling ChatCompletion
- `StartSynthesis` resolves provider by workspace ID before calling ChatCompletion
- Updated both `provider.go` Wire() functions
- Updated synthesis service test stubs to implement `ProviderResolver`

### Workspace provider — expose LLMConfigRepo
- Added `LLMConfigRepo` field to workspace Provider struct
- Wire() now accepts `service.Encryptor` parameter

### Composition root — full wiring
- Added `CANOPY_CRYPTO` config section + cipher initialization
- Created `llmProviderResolver` adapter in `adapters.go`:
  - Resolves workspace-specific LLM provider with cache (sync.RWMutex + map)
  - Falls back to server-wide provider when no workspace config
  - `InvalidateCache()` method for event-driven cache eviction
- Session and synthesis now wired with resolver instead of raw provider

## Files created
- `pkg/crypto/crypto.go`
- `pkg/crypto/config.go`
- `internal/workspace/domain/llm_config.go`
- `internal/workspace/adapter/postgres/migrations/002_add_llm_configs.sql`
- `internal/workspace/adapter/postgres/llm_config_mapper.go`
- `internal/workspace/adapter/postgres/llm_config_repository.go`
- `pkg/llm/resolver.go`

## Files modified
- `internal/workspace/domain/repository.go`
- `internal/workspace/domain/events.go`
- `internal/workspace/adapter/postgres/queries.sql`
- `internal/workspace/service/service.go`
- `internal/workspace/interface/http/v1/handler.go`
- `internal/workspace/interface/http/v1/request.go`
- `internal/workspace/interface/http/v1/response.go`
- `internal/workspace/provider.go`
- `internal/session/service/service.go`
- `internal/session/provider.go`
- `internal/synthesis/service/service.go`
- `internal/synthesis/provider.go`
- `internal/synthesis/service/service_test.go`
- `cmd/server/main.go`
- `cmd/server/adapters.go`

## Decisions made
- AES-256-GCM for API key encryption — authenticated encryption, nonce prepended to ciphertext
- Encryption key loaded from env var (hex-encoded) — no key management service for MVP
- LLM config is per-workspace, not per-user — simplifies access control and caching
- ProviderResolver is a separate interface from Provider — clean separation of concerns
- Resolver caches providers in memory (map + RWMutex) — invalidated via events
- Server-wide LLM config becomes fallback, not override — workspace config always wins when present
- API key masked in GET response (first 3 + last 4 chars) — never expose full key over the wire

## Open questions
- Key rotation (re-encrypt all stored keys with new master key)
- Should resolver validate API key on save (health check call to provider)?
- Per-user API keys (currently per-workspace only)
- Rate limiting per workspace
- Usage tracking / billing per workspace LLM config

## Next steps
- Apply migration `002_add_llm_configs.sql`
- Event subscriber for cache invalidation (LLM config updated/deleted → resolver.InvalidateCache)
- Smoke test new endpoints
- Node x/y coordinates for canvas view
