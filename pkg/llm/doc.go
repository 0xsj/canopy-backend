// Package llm defines the port and adapters for LLM provider integration.
//
// This package is a placeholder — implementation comes when the session,
// synthesis, or notification bounded contexts are built (they are the consumers).
//
// # Purpose
//
// Every AI interaction in Canopy flows through a single LLM provider port.
// The port abstracts provider-specific details so that domain logic never
// couples to a particular vendor's API. Adapters translate between the
// domain's generic request format and each provider's wire protocol.
//
// # Port (the interface)
//
// The provider port defines three operations:
//
//   - ChatCompletion — the core prompting interaction. Accepts a message
//     history and returns a model response. Used by the session context
//     for the two-phase leaf creation flow (exploration → shaping).
//
//   - StructuredOutput — produces a typed leaf structure from conversation
//     context. The model response is parsed into the leaf schema (title,
//     summary, key points, open questions, tags). Used at session confirmation.
//
//   - Synthesis — combines multiple leaf structures into a new one with
//     source attribution. Used by the synthesis context when a user selects
//     leaves to merge.
//
// Each operation accepts a provider configuration containing the model
// identifier, API key, and provider-specific parameters. This is what
// enables BYOK (bring your own key).
//
// # Adapters (planned)
//
//   - Anthropic — Claude models via the Anthropic API
//   - OpenAI — GPT models via the OpenAI API
//   - Others as demand requires
//
// Each adapter handles: API formatting, token counting, rate limiting,
// error normalization, and streaming (if supported). Adapters are stateless
// — provider configuration is passed per-call, not stored in the adapter.
//
// # Platform-Provided vs BYOK
//
// Users without their own API key use the platform-provided LLM, which
// routes through a default adapter configuration using Canopy's own keys.
// Usage is metered and rate-limited.
//
// BYOK users provide their own API key and preferred model. The system
// routes their sessions through the corresponding adapter. Keys are
// encrypted at rest, never logged, and never included in domain events.
//
// # Context Assembly
//
// Context assembly is NOT part of this package. It lives in the session
// context as a separate port (ContextAssembler), since assembly strategy
// varies by session type (exploration, shaping, synthesis, digest).
// This package only handles the LLM call itself.
//
// # Consumers
//
//   - internal/session — exploration and shaping prompting flows
//   - internal/synthesis — multi-leaf synthesis workflow
//   - internal/notification — AI-generated digest summaries
//
// # Design Sketch
//
//	type Provider interface {
//	    ChatCompletion(ctx context.Context, req ChatRequest) (ChatResponse, error)
//	    StructuredOutput(ctx context.Context, req StructuredRequest) (StructuredResponse, error)
//	    Synthesis(ctx context.Context, req SynthesisRequest) (SynthesisResponse, error)
//	}
//
//	type ChatRequest struct {
//	    Model    string
//	    APIKey   string
//	    Messages []Message
//	    Options  Options  // temperature, max tokens, etc.
//	}
//
//	type Message struct {
//	    Role    string // "system", "user", "assistant"
//	    Content string
//	}
//
// # Config
//
// Follows the standard config.Section pattern:
//
//	CANOPY_LLM_DEFAULT_PROVIDER  — "anthropic" or "openai"
//	CANOPY_LLM_DEFAULT_MODEL     — e.g. "claude-sonnet-4-5-20250929"
//	CANOPY_LLM_ANTHROPIC_API_KEY — platform key for Anthropic
//	CANOPY_LLM_OPENAI_API_KEY    — platform key for OpenAI
//	CANOPY_LLM_REQUEST_TIMEOUT   — per-call timeout (default "60s")
package llm
