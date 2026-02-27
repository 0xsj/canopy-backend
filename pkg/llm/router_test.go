package llm

import "testing"

func TestModelRouter_DefaultTierRouting(t *testing.T) {
	r := NewModelRouter()

	tests := []struct {
		provider string
		task     TaskType
		want     string
	}{
		// Anthropic
		{"anthropic", TaskSynthesis, "claude-opus-4-20250918"},
		{"anthropic", TaskConvergence, "claude-opus-4-20250918"},
		{"anthropic", TaskExploration, "claude-sonnet-4-20250514"},
		{"anthropic", TaskShaping, "claude-sonnet-4-20250514"},
		{"anthropic", TaskDigest, "claude-sonnet-4-20250514"},
		{"anthropic", TaskSummarize, "claude-haiku-3-5-20241022"},
		{"anthropic", TaskClassify, "claude-haiku-3-5-20241022"},

		// OpenAI
		{"openai", TaskSynthesis, "gpt-4o"},
		{"openai", TaskExploration, "gpt-4o-mini"},
		{"openai", TaskSummarize, "gpt-4o-mini"},

		// Gemini
		{"gemini", TaskSynthesis, "gemini-1.5-pro"},
		{"gemini", TaskExploration, "gemini-1.5-flash"},
		{"gemini", TaskClassify, "gemini-1.5-flash-8b"},
	}

	for _, tt := range tests {
		got := r.Resolve(tt.provider, tt.task, "fallback")
		if got != tt.want {
			t.Errorf("Resolve(%q, %q) = %q, want %q", tt.provider, tt.task, got, tt.want)
		}
	}
}

func TestModelRouter_OverrideTakesPrecedence(t *testing.T) {
	r := NewModelRouter()
	r.SetOverride("anthropic", TaskExploration, "claude-opus-4-20250918")

	got := r.Resolve("anthropic", TaskExploration, "fallback")
	if got != "claude-opus-4-20250918" {
		t.Errorf("Resolve with override = %q, want claude-opus-4-20250918", got)
	}

	// Other tasks unaffected.
	got = r.Resolve("anthropic", TaskShaping, "fallback")
	if got != "claude-sonnet-4-20250514" {
		t.Errorf("Resolve without override = %q, want claude-sonnet-4-20250514", got)
	}
}

func TestModelRouter_SetDefaultReplacesTier(t *testing.T) {
	r := NewModelRouter()
	r.SetDefault("anthropic", TierBalanced, "claude-sonnet-4-5-20250929")

	got := r.Resolve("anthropic", TaskExploration, "fallback")
	if got != "claude-sonnet-4-5-20250929" {
		t.Errorf("Resolve after SetDefault = %q, want claude-sonnet-4-5-20250929", got)
	}
}

func TestModelRouter_UnknownProviderReturnsFallback(t *testing.T) {
	r := NewModelRouter()

	got := r.Resolve("cohere", TaskExploration, "my-fallback-model")
	if got != "my-fallback-model" {
		t.Errorf("Resolve(unknown provider) = %q, want my-fallback-model", got)
	}
}

func TestModelRouter_SetDefaultNewProvider(t *testing.T) {
	r := NewModelRouter()
	r.SetDefault("mistral", TierBalanced, "mistral-large")

	got := r.Resolve("mistral", TaskExploration, "fallback")
	if got != "mistral-large" {
		t.Errorf("Resolve(new provider) = %q, want mistral-large", got)
	}
}

func TestModelRouter_OverrideOnNewProvider(t *testing.T) {
	r := NewModelRouter()
	r.SetOverride("mistral", TaskClassify, "mistral-small")

	got := r.Resolve("mistral", TaskClassify, "fallback")
	if got != "mistral-small" {
		t.Errorf("Resolve(override on new provider) = %q, want mistral-small", got)
	}

	// No default for other tasks — should fall back.
	got = r.Resolve("mistral", TaskSynthesis, "fallback")
	if got != "fallback" {
		t.Errorf("Resolve(no default) = %q, want fallback", got)
	}
}
