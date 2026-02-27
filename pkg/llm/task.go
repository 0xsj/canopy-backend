package llm

// TaskType identifies the kind of LLM work being performed.
// Services pass a TaskType when resolving a provider so the system
// can route to the appropriate model tier.
type TaskType string

const (
	// TaskExploration is interactive brainstorming with the user.
	TaskExploration TaskType = "exploration"

	// TaskShaping is refining ideas into structured leaves.
	TaskShaping TaskType = "shaping"

	// TaskSynthesis combines multiple leaves into a coherent whole.
	TaskSynthesis TaskType = "synthesis"

	// TaskDigest summarizes recent workspace activity.
	TaskDigest TaskType = "digest"

	// TaskConvergence finds consensus across contributions.
	TaskConvergence TaskType = "convergence"

	// TaskSummarize generates short summaries (titles, one-liners).
	TaskSummarize TaskType = "summarize"

	// TaskClassify categorizes or tags content.
	TaskClassify TaskType = "classify"
)

// ModelTier represents a class of model capability.
// Higher tiers are more capable but slower and more expensive.
type ModelTier string

const (
	// TierFlagship is the most capable tier — complex reasoning, synthesis,
	// convergence. Slower and most expensive.
	TierFlagship ModelTier = "flagship"

	// TierBalanced is good capability with reasonable speed — interactive
	// chat, exploration, shaping.
	TierBalanced ModelTier = "balanced"

	// TierFast is basic capability, very fast — summarization, classification,
	// title generation.
	TierFast ModelTier = "fast"
)

// DefaultTier returns the default model tier for a given task type.
// This encodes the core routing logic: complex tasks get flagship models,
// interactive tasks get balanced, and simple extraction gets fast.
func DefaultTier(task TaskType) ModelTier {
	switch task {
	case TaskSynthesis, TaskConvergence:
		return TierFlagship
	case TaskExploration, TaskShaping, TaskDigest:
		return TierBalanced
	case TaskSummarize, TaskClassify:
		return TierFast
	default:
		return TierBalanced
	}
}
