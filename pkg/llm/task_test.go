package llm

import "testing"

func TestDefaultTier_FlagshipTasks(t *testing.T) {
	for _, task := range []TaskType{TaskSynthesis, TaskConvergence} {
		if tier := DefaultTier(task); tier != TierFlagship {
			t.Errorf("DefaultTier(%q) = %q, want %q", task, tier, TierFlagship)
		}
	}
}

func TestDefaultTier_BalancedTasks(t *testing.T) {
	for _, task := range []TaskType{TaskExploration, TaskShaping, TaskDigest} {
		if tier := DefaultTier(task); tier != TierBalanced {
			t.Errorf("DefaultTier(%q) = %q, want %q", task, tier, TierBalanced)
		}
	}
}

func TestDefaultTier_FastTasks(t *testing.T) {
	for _, task := range []TaskType{TaskSummarize, TaskClassify} {
		if tier := DefaultTier(task); tier != TierFast {
			t.Errorf("DefaultTier(%q) = %q, want %q", task, tier, TierFast)
		}
	}
}

func TestDefaultTier_UnknownTaskDefaultsToBalanced(t *testing.T) {
	if tier := DefaultTier("something_new"); tier != TierBalanced {
		t.Errorf("DefaultTier(unknown) = %q, want %q", tier, TierBalanced)
	}
}

func TestDefaultTier_EmptyTaskDefaultsToBalanced(t *testing.T) {
	if tier := DefaultTier(""); tier != TierBalanced {
		t.Errorf("DefaultTier(\"\") = %q, want %q", tier, TierBalanced)
	}
}
