package llm

import (
	"strings"
	"testing"
)

func TestEstimateTokens_EmptyString(t *testing.T) {
	if got := EstimateTokens(""); got != 0 {
		t.Errorf("EstimateTokens(\"\") = %d, want 0", got)
	}
}

func TestEstimateTokens_SingleWord(t *testing.T) {
	got := EstimateTokens("hello")
	// 1 word * 4/3 ≈ 1 (integer math: (1*4+2)/3 = 2)
	if got < 1 || got > 2 {
		t.Errorf("EstimateTokens(\"hello\") = %d, want 1-2", got)
	}
}

func TestEstimateTokens_ThreeWords(t *testing.T) {
	got := EstimateTokens("hello world foo")
	// 3 words → (3*4+2)/3 = 4
	if got != 4 {
		t.Errorf("EstimateTokens(3 words) = %d, want 4", got)
	}
}

func TestEstimateTokens_LongerText(t *testing.T) {
	// 100 words should estimate ~133 tokens
	words := make([]string, 100)
	for i := range words {
		words[i] = "word"
	}
	text := strings.Join(words, " ")
	got := EstimateTokens(text)
	if got < 130 || got > 140 {
		t.Errorf("EstimateTokens(100 words) = %d, want ~133", got)
	}
}

func TestEstimateTokens_WhitespaceOnly(t *testing.T) {
	if got := EstimateTokens("   \t\n  "); got != 0 {
		t.Errorf("EstimateTokens(whitespace) = %d, want 0", got)
	}
}

func TestEstimateMessagesTokens_Empty(t *testing.T) {
	if got := EstimateMessagesTokens(nil); got != 0 {
		t.Errorf("EstimateMessagesTokens(nil) = %d, want 0", got)
	}
}

func TestEstimateMessagesTokens_IncludesOverhead(t *testing.T) {
	msgs := []Message{
		{Role: "system", Content: "hello world foo"},   // 4 tokens + 4 overhead
		{Role: "user", Content: "hello world foo bar"}, // ~5 tokens + 4 overhead
	}
	got := EstimateMessagesTokens(msgs)
	contentOnly := EstimateTokens("hello world foo") + EstimateTokens("hello world foo bar")
	overhead := 4 * 2 // 4 per message
	want := contentOnly + overhead
	if got != want {
		t.Errorf("EstimateMessagesTokens = %d, want %d (content %d + overhead %d)", got, want, contentOnly, overhead)
	}
}

func TestTruncateToTokenBudget_FitsWithinBudget(t *testing.T) {
	text := "short text"
	got := TruncateToTokenBudget(text, 1000)
	if got != text {
		t.Errorf("TruncateToTokenBudget = %q, want %q (should not truncate)", got, text)
	}
}

func TestTruncateToTokenBudget_TruncatesLongText(t *testing.T) {
	words := make([]string, 100)
	for i := range words {
		words[i] = "word"
	}
	text := strings.Join(words, " ")

	// Budget of 20 tokens → ~15 words (20*3/4)
	got := TruncateToTokenBudget(text, 20)
	resultWords := strings.Fields(strings.TrimSuffix(got, "..."))
	if len(resultWords) != 15 {
		t.Errorf("truncated to %d words, want 15", len(resultWords))
	}
	if !strings.HasSuffix(got, "...") {
		t.Error("truncated text should end with ...")
	}
}

func TestTruncateToTokenBudget_ZeroBudget(t *testing.T) {
	if got := TruncateToTokenBudget("hello world", 0); got != "" {
		t.Errorf("TruncateToTokenBudget(0) = %q, want empty", got)
	}
}

func TestTruncateToTokenBudget_NegativeBudget(t *testing.T) {
	if got := TruncateToTokenBudget("hello world", -5); got != "" {
		t.Errorf("TruncateToTokenBudget(-5) = %q, want empty", got)
	}
}

func TestTruncateToTokenBudget_EmptyText(t *testing.T) {
	if got := TruncateToTokenBudget("", 100); got != "" {
		t.Errorf("TruncateToTokenBudget(\"\") = %q, want empty", got)
	}
}
