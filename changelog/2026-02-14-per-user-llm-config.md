# 2026-02-14 — Per-User LLM Config (BYOM Extension)

## What was done
- Added `UserLLMConfig` entity to the identity domain with `LLMProvider` type, validation, and constructor/reconstruct pattern
- Added `UserLLMConfigRepository` port to identity domain
- Added 2 event types: `identity.user.llm_config.updated` and `identity.user.llm_config.deleted`
- Created migration `002_add_user_llm_configs.sql` — `user_llm_configs` table with FK to users, applied to live DB
- Added 3 sqlc queries (upsert, find by user, delete) and generated code
- Created mapper and repository adapter following existing workspace LLM config patterns
- Added `Encryptor` interface and 3 service methods (`SetLLMConfig`, `GetLLMConfig`, `DeleteLLMConfig`) to identity service
- Added 3 HTTP endpoints: `PUT/GET/DELETE /api/v1/users/me/llm-config`
- Updated identity provider to accept `Encryptor` and expose `LLMConfigRepo`
- Upgraded `llmProviderResolver` to support User > Workspace > Server hierarchy
  - Composite cache key: `u:<userID>:w:<wsID>` for user-level, `w:<wsID>` for workspace-level
  - `InvalidateUserCache(userID)` clears all cache entries for a user
- Added `userLLMConfigHandler` subscriber for cache invalidation (filters `workspace.>` by event type)
- Added 2 domain audit registry entries for user LLM config events
- Updated composition root to pass cipher to identity provider and user config repo to resolver

## Decisions made
- Per-user LLM config lives in the identity context (it's a personal preference, not workspace-scoped)
- Cache key uses composite format `u:<id>:w:<id>` to support per-user-per-workspace caching
- User LLM config events publish with empty workspaceID (identity context is not workspace-scoped), so subscriber uses `workspace.>` with event type filtering instead of a specific subject pattern
- Resolver tries user config first, falls through to workspace, then server fallback — no auth means skip user check

## Open questions
- None

## Next steps
- Smoke test the 3 user LLM config endpoints
- Verify session with user config uses user's key instead of workspace/server
- Write domain tests for `UserLLMConfig` entity
