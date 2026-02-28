package llm

import "strings"

// EstimateTokens returns an approximate token count for a string.
// Uses the heuristic: 1 word ≈ 1.33 tokens. This is a rough estimate
// that works reasonably well across English text for both GPT and Claude
// tokenizers. Good enough for budget decisions — not for billing.
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}
	words := len(strings.Fields(text))
	// Multiply by 4/3 (≈1.33) using integer math to avoid float.
	return (words*4 + 2) / 3
}

// EstimateMessagesTokens returns the total estimated token count
// across a slice of messages, including a small per-message overhead
// for role markers and formatting.
func EstimateMessagesTokens(messages []Message) int {
	total := 0
	for _, m := range messages {
		total += EstimateTokens(m.Content)
		total += 4 // per-message overhead (role, delimiters)
	}
	return total
}

// TruncateToTokenBudget trims text to fit within an approximate token budget.
// Returns the original text if it fits, otherwise truncates at a word boundary.
func TruncateToTokenBudget(text string, budget int) string {
	if budget <= 0 {
		return ""
	}
	if EstimateTokens(text) <= budget {
		return text
	}

	// Target word count: budget * 3/4 (inverse of the 4/3 multiplier).
	targetWords := (budget * 3) / 4
	if targetWords <= 0 {
		return ""
	}

	words := strings.Fields(text)
	if len(words) <= targetWords {
		return text
	}

	return strings.Join(words[:targetWords], " ") + "..."
}
