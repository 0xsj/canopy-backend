package llm

// ModelRouter resolves a concrete model name given a provider and task type.
// It uses DefaultTier to map task → tier, then looks up the model for that
// provider + tier combination. Overrides take precedence over defaults.
type ModelRouter struct {
	// defaults maps provider → tier → model name.
	defaults map[string]map[ModelTier]string

	// overrides maps provider → task → model name.
	// When set, bypasses tier lookup entirely for that specific task.
	overrides map[string]map[TaskType]string
}

// NewModelRouter creates a router with sensible defaults for each provider.
func NewModelRouter() *ModelRouter {
	return &ModelRouter{
		defaults: map[string]map[ModelTier]string{
			"anthropic": {
				TierFlagship: "claude-opus-4-20250918",
				TierBalanced: "claude-sonnet-4-20250514",
				TierFast:     "claude-haiku-3-5-20241022",
			},
			"openai": {
				TierFlagship: "gpt-4o",
				TierBalanced: "gpt-4o-mini",
				TierFast:     "gpt-4o-mini",
			},
			"gemini": {
				TierFlagship: "gemini-1.5-pro",
				TierBalanced: "gemini-1.5-flash",
				TierFast:     "gemini-1.5-flash-8b",
			},
		},
		overrides: make(map[string]map[TaskType]string),
	}
}

// SetDefault sets or replaces the default model for a provider + tier.
func (r *ModelRouter) SetDefault(provider string, tier ModelTier, model string) {
	if r.defaults[provider] == nil {
		r.defaults[provider] = make(map[ModelTier]string)
	}
	r.defaults[provider][tier] = model
}

// SetOverride sets a per-task model override for a provider.
// This bypasses tier-based routing for that specific task.
func (r *ModelRouter) SetOverride(provider string, task TaskType, model string) {
	if r.overrides[provider] == nil {
		r.overrides[provider] = make(map[TaskType]string)
	}
	r.overrides[provider][task] = model
}

// Resolve returns the concrete model name for a provider + task combination.
// Resolution order:
//  1. Per-task override (if set)
//  2. Tier default for the provider
//  3. The fallback model (passed in, typically the server config model)
func (r *ModelRouter) Resolve(provider string, task TaskType, fallback string) string {
	// Check per-task override first.
	if tasks, ok := r.overrides[provider]; ok {
		if model, ok := tasks[task]; ok {
			return model
		}
	}

	// Look up by tier.
	tier := DefaultTier(task)
	if tiers, ok := r.defaults[provider]; ok {
		if model, ok := tiers[tier]; ok {
			return model
		}
	}

	return fallback
}
