package main

import (
	"context"
	"strings"
	"testing"

	expdomain "github.com/0xsj/canopy-backend/internal/exploration/domain"
	seeddomain "github.com/0xsj/canopy-backend/internal/seed/domain"
	sessiondomain "github.com/0xsj/canopy-backend/internal/session/domain"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// --- Test doubles ---

type stubSeedFinder struct {
	seed seeddomain.Seed
	err  error
}

func (s *stubSeedFinder) FindByID(_ context.Context, _ types.SeedID) (seeddomain.Seed, error) {
	return s.seed, s.err
}

type stubLeafFinder struct {
	leaf   expdomain.Leaf
	leaves []expdomain.Leaf
	err    error
}

func (s *stubLeafFinder) FindByID(_ context.Context, _ types.LeafID) (expdomain.Leaf, error) {
	return s.leaf, s.err
}

func (s *stubLeafFinder) FindByIDs(_ context.Context, _ []types.LeafID) ([]expdomain.Leaf, error) {
	return s.leaves, s.err
}

// --- Helpers ---

func testSeed() seeddomain.Seed {
	return seeddomain.ReconstructSeed(
		types.SeedIDFrom("seed_test"),
		types.WorkspaceIDFrom("ws_test"),
		types.UserIDFrom("user_test"),
		"What is distributed consensus?",
		"Explore approaches to distributed consensus in decentralized systems.",
		map[string]any{"scope": "technical", "depth": "intermediate"},
		[]string{"distributed-systems", "consensus"},
		0, 0,
		types.NewMutableTimestamps(),
	)
}

func testLeaf() expdomain.Leaf {
	return expdomain.ReconstructLeaf(
		types.LeafIDFrom("leaf_test"),
		types.WorkspaceIDFrom("ws_test"),
		types.SeedIDFrom("seed_test"),
		types.BranchIDFrom("branch_test"),
		types.UserIDFrom("user_test"),
		types.LeafID{}, // no parent
		"Raft vs Paxos",
		"Raft is designed for understandability while Paxos optimizes for correctness proofs.",
		[]string{"Raft uses a strong leader model", "Paxos is more flexible but harder to implement"},
		[]string{"How does PBFT compare?", "What about leaderless approaches?"},
		[]string{"consensus", "raft", "paxos"},
		expdomain.LayerUnderstory,
		nil, // no sources
		nil, // no metadata
		0, 0,
		types.NewTimestamps(),
	)
}

func testSession(sessionType sessiondomain.SessionType, parentLeafID types.LeafID, messages []sessiondomain.Message) sessiondomain.Session {
	return testSessionWithSources(sessionType, parentLeafID, nil, messages)
}

func testSessionWithSources(sessionType sessiondomain.SessionType, parentLeafID types.LeafID, sourceLeafIDs []types.LeafID, messages []sessiondomain.Message) sessiondomain.Session {
	return sessiondomain.ReconstructSession(
		sessiondomain.SessionIDFrom("ses_test"),
		types.WorkspaceIDFrom("ws_test"),
		types.UserIDFrom("user_test"),
		types.SeedIDFrom("seed_test"),
		parentLeafID,
		sourceLeafIDs,
		sessionType,
		sessiondomain.StatusActive,
		messages,
		types.NewMutableTimestamps(),
	)
}

// --- Tests ---

func TestContextAssembler_Exploration_WithSeedOnly(t *testing.T) {
	a := &contextAssembler{
		seeds:  &stubSeedFinder{seed: testSeed()},
		leaves: &stubLeafFinder{},
		log:    logger.NewNoop(),
	}

	session := testSession(sessiondomain.SessionExploration, types.LeafID{}, []sessiondomain.Message{
		{Role: "user", Content: "Tell me about consensus algorithms"},
	})

	messages, err := a.Assemble(context.Background(), session)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("len(messages) = %d, want 2 (system + 1 user)", len(messages))
	}

	system := messages[0]
	if system.Role != "system" {
		t.Errorf("messages[0].Role = %q, want system", system.Role)
	}
	if !strings.Contains(system.Content, "What is distributed consensus?") {
		t.Error("system prompt should contain seed title")
	}
	if !strings.Contains(system.Content, "Explore approaches") {
		t.Error("system prompt should contain seed description")
	}
	if !strings.Contains(system.Content, "scope") {
		t.Error("system prompt should contain seed constraints")
	}
	if !strings.Contains(system.Content, "collaborative thinking") {
		t.Error("system prompt should contain exploration instructions")
	}

	if messages[1].Content != "Tell me about consensus algorithms" {
		t.Errorf("messages[1].Content = %q, want user message", messages[1].Content)
	}
}

func TestContextAssembler_Exploration_WithParentLeaf(t *testing.T) {
	a := &contextAssembler{
		seeds:  &stubSeedFinder{seed: testSeed()},
		leaves: &stubLeafFinder{leaf: testLeaf()},
		log:    logger.NewNoop(),
	}

	session := testSession(sessiondomain.SessionExploration, types.LeafIDFrom("leaf_test"), []sessiondomain.Message{
		{Role: "user", Content: "Let's explore PBFT"},
	})

	messages, err := a.Assemble(context.Background(), session)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}

	system := messages[0]
	if !strings.Contains(system.Content, "Raft vs Paxos") {
		t.Error("system prompt should contain parent leaf title")
	}
	if !strings.Contains(system.Content, "Raft is designed for understandability") {
		t.Error("system prompt should contain parent leaf summary")
	}
	if !strings.Contains(system.Content, "strong leader model") {
		t.Error("system prompt should contain parent leaf key points")
	}
	if !strings.Contains(system.Content, "How does PBFT compare") {
		t.Error("system prompt should contain parent leaf open questions")
	}
}

func TestContextAssembler_Shaping(t *testing.T) {
	a := &contextAssembler{
		seeds:  &stubSeedFinder{seed: testSeed()},
		leaves: &stubLeafFinder{},
		log:    logger.NewNoop(),
	}

	session := testSession(sessiondomain.SessionShaping, types.LeafID{}, []sessiondomain.Message{
		{Role: "user", Content: "I want to structure my thoughts on Raft"},
		{Role: "assistant", Content: "Let's start with the title."},
	})

	messages, err := a.Assemble(context.Background(), session)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}

	if len(messages) != 3 {
		t.Fatalf("len(messages) = %d, want 3 (system + 2 conversation)", len(messages))
	}

	system := messages[0]
	if !strings.Contains(system.Content, "refine and structure") {
		t.Error("shaping prompt should contain structuring instructions")
	}
	if !strings.Contains(system.Content, "Title") {
		t.Error("shaping prompt should mention leaf structure fields")
	}
	if !strings.Contains(system.Content, "Key Points") {
		t.Error("shaping prompt should mention key points")
	}
}

func TestContextAssembler_Synthesis(t *testing.T) {
	a := &contextAssembler{
		seeds:  &stubSeedFinder{seed: testSeed()},
		leaves: &stubLeafFinder{},
		log:    logger.NewNoop(),
	}

	session := testSession(sessiondomain.SessionSynthesis, types.LeafID{}, []sessiondomain.Message{
		{Role: "user", Content: "Merge these two ideas about consensus"},
	})

	messages, err := a.Assemble(context.Background(), session)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}

	system := messages[0]
	if !strings.Contains(system.Content, "synthesize") {
		t.Error("synthesis prompt should contain synthesis instructions")
	}
	if !strings.Contains(system.Content, "attribution") {
		t.Error("synthesis prompt should mention attribution")
	}
}

func TestContextAssembler_Digest(t *testing.T) {
	a := &contextAssembler{
		seeds:  &stubSeedFinder{seed: testSeed()},
		leaves: &stubLeafFinder{},
		log:    logger.NewNoop(),
	}

	session := testSession(sessiondomain.SessionDigest, types.LeafID{}, nil)

	messages, err := a.Assemble(context.Background(), session)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("len(messages) = %d, want 1 (system only, no conversation)", len(messages))
	}

	system := messages[0]
	if !strings.Contains(system.Content, "summarizing") {
		t.Error("digest prompt should contain summarization instructions")
	}
}

func TestContextAssembler_PreservesConversationOrder(t *testing.T) {
	a := &contextAssembler{
		seeds:  &stubSeedFinder{seed: testSeed()},
		leaves: &stubLeafFinder{},
		log:    logger.NewNoop(),
	}

	conversation := []sessiondomain.Message{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "second"},
		{Role: "user", Content: "third"},
	}
	session := testSession(sessiondomain.SessionExploration, types.LeafID{}, conversation)

	messages, err := a.Assemble(context.Background(), session)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}

	if len(messages) != 4 {
		t.Fatalf("len(messages) = %d, want 4 (system + 3 conversation)", len(messages))
	}

	// Verify conversation order is preserved after system message.
	for i, want := range conversation {
		got := messages[i+1]
		if got.Role != want.Role || got.Content != want.Content {
			t.Errorf("messages[%d] = {%q, %q}, want {%q, %q}", i+1, got.Role, got.Content, want.Role, want.Content)
		}
	}
}

func TestContextAssembler_SeedNotFound_ReturnsError(t *testing.T) {
	a := &contextAssembler{
		seeds:  &stubSeedFinder{err: context.DeadlineExceeded},
		leaves: &stubLeafFinder{},
		log:    logger.NewNoop(),
	}

	session := testSession(sessiondomain.SessionExploration, types.LeafID{}, nil)

	_, err := a.Assemble(context.Background(), session)
	if err == nil {
		t.Error("Assemble() error = nil, want error when seed not found")
	}
}

func TestContextAssembler_ParentLeafNotFound_ContinuesWithout(t *testing.T) {
	a := &contextAssembler{
		seeds:  &stubSeedFinder{seed: testSeed()},
		leaves: &stubLeafFinder{err: context.DeadlineExceeded},
		log:    logger.NewNoop(),
	}

	session := testSession(sessiondomain.SessionExploration, types.LeafIDFrom("missing_leaf"), []sessiondomain.Message{
		{Role: "user", Content: "Hello"},
	})

	messages, err := a.Assemble(context.Background(), session)
	if err != nil {
		t.Fatalf("Assemble() should not error when parent leaf is missing, got: %v", err)
	}

	// Should still have system + user message, just without parent leaf context.
	if len(messages) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(messages))
	}

	system := messages[0]
	if strings.Contains(system.Content, "Building On") {
		t.Error("system prompt should NOT contain parent leaf section when leaf not found")
	}
}

func TestContextAssembler_EmptyConstraints_OmitsSection(t *testing.T) {
	seedNoConstraints := seeddomain.ReconstructSeed(
		types.SeedIDFrom("seed_test"),
		types.WorkspaceIDFrom("ws_test"),
		types.UserIDFrom("user_test"),
		"Simple topic",
		"",
		nil, // no constraints
		nil,
		0, 0,
		types.NewMutableTimestamps(),
	)

	a := &contextAssembler{
		seeds:  &stubSeedFinder{seed: seedNoConstraints},
		leaves: &stubLeafFinder{},
		log:    logger.NewNoop(),
	}

	session := testSession(sessiondomain.SessionExploration, types.LeafID{}, nil)

	messages, err := a.Assemble(context.Background(), session)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}

	system := messages[0]
	if strings.Contains(system.Content, "Constraints") {
		t.Error("system prompt should not contain Constraints section when there are none")
	}
}

func TestContextAssembler_ImplementsInterface(t *testing.T) {
	var _ sessiondomain.ContextAssembler = (*contextAssembler)(nil)
}

// --- Budget tests ---

func TestBudgetForSessionType_AllTypesReturnNonZero(t *testing.T) {
	types := []sessiondomain.SessionType{
		sessiondomain.SessionExploration,
		sessiondomain.SessionShaping,
		sessiondomain.SessionSynthesis,
		sessiondomain.SessionDigest,
	}
	for _, st := range types {
		b := budgetForSessionType(st)
		if b.SystemPrompt == 0 {
			t.Errorf("budgetForSessionType(%q).SystemPrompt = 0", st)
		}
		if b.History == 0 {
			t.Errorf("budgetForSessionType(%q).History = 0", st)
		}
	}
}

func TestBudgetForSessionType_SynthesisHasLargestLeafBudget(t *testing.T) {
	synth := budgetForSessionType(sessiondomain.SessionSynthesis)
	explore := budgetForSessionType(sessiondomain.SessionExploration)
	if synth.LeafContext <= explore.LeafContext {
		t.Errorf("synthesis leaf budget (%d) should exceed exploration (%d)", synth.LeafContext, explore.LeafContext)
	}
}

func TestBudgetForSessionType_ExplorationHasLargestHistoryBudget(t *testing.T) {
	explore := budgetForSessionType(sessiondomain.SessionExploration)
	synth := budgetForSessionType(sessiondomain.SessionSynthesis)
	if explore.History <= synth.History {
		t.Errorf("exploration history budget (%d) should exceed synthesis (%d)", explore.History, synth.History)
	}
}

func TestBudgetForSessionType_DigestHasNoLeafBudget(t *testing.T) {
	b := budgetForSessionType(sessiondomain.SessionDigest)
	if b.LeafContext != 0 {
		t.Errorf("digest leaf budget = %d, want 0", b.LeafContext)
	}
}

func TestBudgetForSessionType_UnknownTypeReturnsDefault(t *testing.T) {
	b := budgetForSessionType("unknown_type")
	if b.SystemPrompt == 0 || b.History == 0 {
		t.Error("unknown type should return non-zero default budget")
	}
}

// --- Sliding window tests ---

func TestSlidingWindowHistory_EmptyMessages(t *testing.T) {
	result := slidingWindowHistory(nil, 1000)
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

func TestSlidingWindowHistory_AllFitWithinBudget(t *testing.T) {
	msgs := []sessiondomain.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
		{Role: "user", Content: "how are you"},
	}
	result := slidingWindowHistory(msgs, 10000) // huge budget
	if len(result) != 3 {
		t.Errorf("len = %d, want 3 (all should fit)", len(result))
	}
}

func TestSlidingWindowHistory_TrimsOlderMessages(t *testing.T) {
	msgs := []sessiondomain.Message{
		{Role: "user", Content: "this is the first message and it's quite long with many words to use up tokens"},
		{Role: "assistant", Content: "this is the second message also with many words to use up budget tokens too"},
		{Role: "user", Content: "short last"},
	}
	// Give a budget that fits the last 1-2 messages but not all 3.
	// "short last" ≈ 3 tokens + 4 overhead = 7
	// second msg ≈ ~20 tokens + 4 = 24
	// first msg ≈ ~22 tokens + 4 = 26
	// Budget of 35 should fit last 2 but not all 3.
	result := slidingWindowHistory(msgs, 35)
	if len(result) >= 3 {
		t.Errorf("len = %d, want < 3 (should trim oldest)", len(result))
	}
	// Last message must always be preserved.
	if result[len(result)-1].Content != "short last" {
		t.Error("last message should always be preserved")
	}
}

func TestSlidingWindowHistory_AlwaysPreservesLastMessage(t *testing.T) {
	msgs := []sessiondomain.Message{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "second"},
		{Role: "user", Content: "this is a very long message that by itself exceeds any reasonable token budget we might set for testing purposes here"},
	}
	// Budget of 1 — can't even fit the last message, but it should still be kept.
	result := slidingWindowHistory(msgs, 1)
	if len(result) != 1 {
		t.Errorf("len = %d, want 1 (only last message)", len(result))
	}
	if result[0].Content != msgs[2].Content {
		t.Error("should preserve the last message even when over budget")
	}
}

func TestSlidingWindowHistory_ZeroBudgetReturnsAll(t *testing.T) {
	msgs := []sessiondomain.Message{
		{Role: "user", Content: "hello"},
	}
	// Zero budget returns messages as-is (no windowing).
	result := slidingWindowHistory(msgs, 0)
	if len(result) != 1 {
		t.Errorf("len = %d, want 1", len(result))
	}
}

func TestSlidingWindowHistory_PreservesOrder(t *testing.T) {
	msgs := []sessiondomain.Message{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "second"},
		{Role: "user", Content: "third"},
		{Role: "assistant", Content: "fourth"},
		{Role: "user", Content: "fifth"},
	}
	result := slidingWindowHistory(msgs, 10000)
	for i, m := range result {
		if m.Content != msgs[i].Content {
			t.Errorf("result[%d] = %q, want %q", i, m.Content, msgs[i].Content)
		}
	}
}

// --- Integration: assembler applies sliding window ---

func TestContextAssembler_TrimsLongConversation(t *testing.T) {
	a := &contextAssembler{
		seeds:  &stubSeedFinder{seed: testSeed()},
		leaves: &stubLeafFinder{},
		log:    logger.NewNoop(),
	}

	// Build a conversation with 50 messages — way more than any budget.
	var conversation []sessiondomain.Message
	for i := 0; i < 50; i++ {
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		conversation = append(conversation, sessiondomain.Message{
			Role:    role,
			Content: strings.Repeat("word ", 100), // ~133 tokens each
		})
	}

	session := testSession(sessiondomain.SessionExploration, types.LeafID{}, conversation)

	messages, err := a.Assemble(context.Background(), session)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}

	// Should have system message + fewer than 50 conversation messages.
	conversationCount := len(messages) - 1
	if conversationCount >= 50 {
		t.Errorf("expected trimmed conversation, got %d messages (all 50)", conversationCount)
	}
	if conversationCount == 0 {
		t.Error("should preserve at least 1 conversation message")
	}

	// Last conversation message should be the 50th (most recent).
	last := messages[len(messages)-1]
	if last.Content != conversation[49].Content {
		t.Error("most recent message should be preserved")
	}
}
