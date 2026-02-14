// Package llm defines the port and adapters for LLM provider integration.
//
// Every AI interaction in Canopy flows through a single LLM provider port.
// The port abstracts provider-specific details so that domain logic never
// couples to a particular vendor's API. Adapters translate between the
// domain's generic request format and each provider's wire protocol.
//
// # Port
//
// The Provider interface defines a single ChatCompletion operation that
// accepts a message history and returns a model response. Used by the
// session context for the two-phase leaf creation flow (exploration, shaping).
// StructuredOutput and Synthesis are future additions — they are prompt
// engineering on top of ChatCompletion, not separate API calls.
//
// # Adapters
//
//   - Anthropic — Claude models via the Anthropic Messages API
//   - OpenAI — GPT/o-series models via the Chat Completions API
//   - Gemini — Google models via the Generative AI API
//
// Each adapter handles: API formatting, system message extraction (Anthropic
// and Gemini use a separate system param), error normalization, and usage
// tracking. Adapters hold the API key from config; the model can be
// overridden per-request via ChatRequest.Model.
//
// # Factory
//
// NewProvider(ctx, cfg, log) routes to the correct adapter based on
// cfg.Provider ("anthropic", "openai", or "gemini"). Only one provider
// is active at a time.
//
// # Context Assembly
//
// Context assembly is NOT part of this package. It lives in the session
// context as a separate port (ContextAssembler), since assembly strategy
// varies by session type (exploration, shaping, synthesis, digest).
// This package only handles the LLM call itself.
//
// # Config
//
// Follows the standard config.Section pattern:
//
//	CANOPY_LLM_PROVIDER        — "anthropic", "openai", or "gemini"
//	CANOPY_LLM_MODEL           — e.g. "claude-sonnet-4-5-20250929"
//	CANOPY_LLM_API_KEY         — API key for the selected provider
//	CANOPY_LLM_REQUEST_TIMEOUT — per-call timeout (default "60s")
//	CANOPY_LLM_TEMPERATURE     — default temperature (default 0.7)
//	CANOPY_LLM_MAX_TOKENS      — default max tokens (default 4096)
//
// # Consumers
//
//   - internal/session — exploration and shaping prompting flows
//   - internal/synthesis — multi-leaf synthesis workflow (future)
//   - internal/notification — AI-generated digest summaries (future)
package llm
