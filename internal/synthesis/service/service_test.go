package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/0xsj/canopy-backend/internal/synthesis/domain"
	wsdomain "github.com/0xsj/canopy-backend/internal/workspace/domain"
	"github.com/0xsj/canopy-backend/pkg/auth"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/llm"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// --- Test doubles ---

// stubSynthesisRepo implements domain.SynthesisRepository for unit tests.
type stubSynthesisRepo struct {
	createFn        func(ctx context.Context, workflow domain.SynthesisWorkflow) error
	findByIDFn      func(ctx context.Context, id domain.SynthesisID) (domain.SynthesisWorkflow, error)
	findByWSFn      func(ctx context.Context, wsID types.WorkspaceID) ([]domain.SynthesisWorkflow, error)
	updateFn        func(ctx context.Context, workflow domain.SynthesisWorkflow) error
	createdWorkflow domain.SynthesisWorkflow
	updatedWorkflow domain.SynthesisWorkflow
	updateCalls     int
}

func (s *stubSynthesisRepo) Create(ctx context.Context, workflow domain.SynthesisWorkflow) error {
	s.createdWorkflow = workflow
	if s.createFn != nil {
		return s.createFn(ctx, workflow)
	}
	return nil
}

func (s *stubSynthesisRepo) FindByID(ctx context.Context, id domain.SynthesisID) (domain.SynthesisWorkflow, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(ctx, id)
	}
	return domain.SynthesisWorkflow{}, canopyerr.ErrNotFound
}

func (s *stubSynthesisRepo) FindByWorkspace(ctx context.Context, wsID types.WorkspaceID) ([]domain.SynthesisWorkflow, error) {
	if s.findByWSFn != nil {
		return s.findByWSFn(ctx, wsID)
	}
	return nil, nil
}

func (s *stubSynthesisRepo) Update(ctx context.Context, workflow domain.SynthesisWorkflow) error {
	s.updatedWorkflow = workflow
	s.updateCalls++
	if s.updateFn != nil {
		return s.updateFn(ctx, workflow)
	}
	return nil
}

// stubSourceLeafReader implements SourceLeafReader for unit tests.
type stubSourceLeafReader struct {
	findByIDsFn func(ctx context.Context, ids []types.LeafID) ([]SourceLeaf, error)
}

func (s *stubSourceLeafReader) FindByIDs(ctx context.Context, ids []types.LeafID) ([]SourceLeaf, error) {
	if s.findByIDsFn != nil {
		return s.findByIDsFn(ctx, ids)
	}
	return nil, nil
}

// stubSeedReader implements SeedReader for unit tests.
type stubSeedReader struct {
	findByIDFn func(ctx context.Context, id types.SeedID) (SeedInfo, error)
}

func (s *stubSeedReader) FindByID(ctx context.Context, id types.SeedID) (SeedInfo, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(ctx, id)
	}
	return SeedInfo{}, nil
}

// stubLeafCreator implements SynthesisLeafCreator for unit tests.
type stubLeafCreator struct {
	createFn    func(ctx context.Context, params SynthesisLeafParams) (types.LeafID, error)
	lastParams  SynthesisLeafParams
	createCalls int
}

func (s *stubLeafCreator) Create(ctx context.Context, params SynthesisLeafParams) (types.LeafID, error) {
	s.lastParams = params
	s.createCalls++
	if s.createFn != nil {
		return s.createFn(ctx, params)
	}
	return types.LeafIDFrom("leaf_result"), nil
}

// stubLLMProvider implements llm.Provider for unit tests.
type stubLLMProvider struct {
	chatCompletionFn func(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error)
	lastRequest      llm.ChatRequest
}

func (s *stubLLMProvider) ChatCompletion(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
	s.lastRequest = req
	if s.chatCompletionFn != nil {
		return s.chatCompletionFn(ctx, req)
	}
	return llm.ChatResponse{}, nil
}

// Resolve implements llm.ProviderResolver, returning the stub as the provider.
func (s *stubLLMProvider) Resolve(_ context.Context, _ types.WorkspaceID) (llm.Provider, error) {
	return s, nil
}

// stubWSMembers implements WorkspaceMemberReader for unit tests.
type stubWSMembers struct {
	findMemberFn func(ctx context.Context, wsID types.WorkspaceID, userID types.UserID) (wsdomain.WorkspaceMember, error)
}

func (s *stubWSMembers) FindMember(ctx context.Context, wsID types.WorkspaceID, userID types.UserID) (wsdomain.WorkspaceMember, error) {
	if s.findMemberFn != nil {
		return s.findMemberFn(ctx, wsID, userID)
	}
	return wsdomain.WorkspaceMember{}, nil
}

// stubPublisher implements events.Publisher for unit tests.
type stubPublisher struct {
	publishFn    func(ctx context.Context, event events.Event) error
	published    []events.Event
	publishCalls int
}

func (s *stubPublisher) Publish(ctx context.Context, event events.Event) error {
	s.published = append(s.published, event)
	s.publishCalls++
	if s.publishFn != nil {
		return s.publishFn(ctx, event)
	}
	return nil
}

// --- Interface conformance checks ---

var _ domain.SynthesisRepository = (*stubSynthesisRepo)(nil)
var _ SourceLeafReader = (*stubSourceLeafReader)(nil)
var _ SeedReader = (*stubSeedReader)(nil)
var _ SynthesisLeafCreator = (*stubLeafCreator)(nil)
var _ llm.ProviderResolver = (*stubLLMProvider)(nil)
var _ WorkspaceMemberReader = (*stubWSMembers)(nil)
var _ events.Publisher = (*stubPublisher)(nil)

// --- Test helpers ---

func testCtx(userID string) context.Context {
	claims := auth.Claims{Subject: userID, Issuer: "test"}
	return auth.WithClaims(context.Background(), claims)
}

func testCtxNoAuth() context.Context {
	return context.Background()
}

func validLLMJSON() string {
	result := synthesisResult{
		Title:         "Synthesized Insight",
		Summary:       "A unified understanding of the source ideas.",
		KeyPoints:     []string{"Point A", "Point B", "Point C"},
		OpenQuestions: []string{"What about X?"},
		Tags:          []string{"synthesis", "insight"},
	}
	b, _ := json.Marshal(result)
	return string(b)
}

func defaultSourceLeaves() []SourceLeaf {
	return []SourceLeaf{
		{
			ID:            types.LeafIDFrom("leaf_1"),
			SeedID:        types.SeedIDFrom("seed_test"),
			Title:         "First Idea",
			Summary:       "Summary of the first idea.",
			KeyPoints:     []string{"KP1", "KP2"},
			OpenQuestions: []string{"OQ1"},
			Tags:          []string{"tag1"},
		},
		{
			ID:            types.LeafIDFrom("leaf_2"),
			SeedID:        types.SeedIDFrom("seed_test"),
			Title:         "Second Idea",
			Summary:       "Summary of the second idea.",
			KeyPoints:     []string{"KP3"},
			OpenQuestions: []string{"OQ2", "OQ3"},
			Tags:          []string{"tag2"},
		},
	}
}

func newTestService(
	repo *stubSynthesisRepo,
	llmProv *stubLLMProvider,
	leaves *stubSourceLeafReader,
	seeds *stubSeedReader,
	leafCreator *stubLeafCreator,
	wsMembers *stubWSMembers,
	pub *stubPublisher,
) *Service {
	return New(repo, llmProv, leaves, seeds, leafCreator, wsMembers, pub, logger.NewNoop())
}

// --- parseSynthesisResponse tests ---

func TestParseSynthesisResponse_ValidJSON(t *testing.T) {
	input := `{"title":"My Title","summary":"My Summary","key_points":["A","B"],"open_questions":["Q1"],"tags":["t1"]}`

	result, err := parseSynthesisResponse(input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Title != "My Title" {
		t.Errorf("expected title 'My Title', got %q", result.Title)
	}
	if result.Summary != "My Summary" {
		t.Errorf("expected summary 'My Summary', got %q", result.Summary)
	}
	if len(result.KeyPoints) != 2 {
		t.Errorf("expected 2 key points, got %d", len(result.KeyPoints))
	}
	if len(result.OpenQuestions) != 1 {
		t.Errorf("expected 1 open question, got %d", len(result.OpenQuestions))
	}
	if len(result.Tags) != 1 {
		t.Errorf("expected 1 tag, got %d", len(result.Tags))
	}
}

func TestParseSynthesisResponse_JSONWithMarkdownFences(t *testing.T) {
	input := "```json\n{\"title\":\"Fenced Title\",\"summary\":\"Fenced Summary\",\"key_points\":[],\"open_questions\":[],\"tags\":[]}\n```"

	result, err := parseSynthesisResponse(input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Title != "Fenced Title" {
		t.Errorf("expected title 'Fenced Title', got %q", result.Title)
	}
	if result.Summary != "Fenced Summary" {
		t.Errorf("expected summary 'Fenced Summary', got %q", result.Summary)
	}
}

func TestParseSynthesisResponse_JSONWithLeadingText(t *testing.T) {
	input := "Here is the result:\n{\"title\":\"Extracted\",\"summary\":\"From mixed content\",\"key_points\":[],\"open_questions\":[],\"tags\":[]}\nSome trailing text."

	result, err := parseSynthesisResponse(input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Title != "Extracted" {
		t.Errorf("expected title 'Extracted', got %q", result.Title)
	}
}

func TestParseSynthesisResponse_MissingTitle(t *testing.T) {
	input := `{"title":"","summary":"Has summary","key_points":[],"open_questions":[],"tags":[]}`

	_, err := parseSynthesisResponse(input)
	if err == nil {
		t.Fatal("expected error for missing title, got nil")
	}
	if !strings.Contains(err.Error(), "missing title") {
		t.Errorf("expected error about missing title, got: %v", err)
	}
}

func TestParseSynthesisResponse_MissingSummary(t *testing.T) {
	input := `{"title":"Has Title","summary":"","key_points":[],"open_questions":[],"tags":[]}`

	_, err := parseSynthesisResponse(input)
	if err == nil {
		t.Fatal("expected error for missing summary, got nil")
	}
	if !strings.Contains(err.Error(), "missing summary") {
		t.Errorf("expected error about missing summary, got: %v", err)
	}
}

func TestParseSynthesisResponse_InvalidJSON(t *testing.T) {
	input := "this is not valid JSON at all"

	_, err := parseSynthesisResponse(input)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("expected error about invalid JSON, got: %v", err)
	}
}

func TestParseSynthesisResponse_EmptyContent(t *testing.T) {
	_, err := parseSynthesisResponse("")
	if err == nil {
		t.Fatal("expected error for empty content, got nil")
	}
}

func TestParseSynthesisResponse_OnlyWhitespace(t *testing.T) {
	_, err := parseSynthesisResponse("   \n\t  ")
	if err == nil {
		t.Fatal("expected error for whitespace-only content, got nil")
	}
}

func TestParseSynthesisResponse_PartialJSON(t *testing.T) {
	// Missing closing brace — should fail to parse.
	input := `{"title":"Partial","summary":"Data","key_points":[]`

	_, err := parseSynthesisResponse(input)
	if err == nil {
		t.Fatal("expected error for partial JSON, got nil")
	}
}

func TestParseSynthesisResponse_NullKeyPoints(t *testing.T) {
	input := `{"title":"Title","summary":"Summary","key_points":null,"open_questions":null,"tags":null}`

	result, err := parseSynthesisResponse(input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.KeyPoints != nil {
		t.Errorf("expected nil key_points, got %v", result.KeyPoints)
	}
}

// --- buildSynthesisLLMPrompt tests ---

func TestBuildSynthesisLLMPrompt_WithDescription(t *testing.T) {
	seed := SeedInfo{
		Title:       "Test Seed",
		Description: "A description of the seed topic.",
	}
	leaves := defaultSourceLeaves()

	prompt := buildSynthesisLLMPrompt(seed, leaves)

	if !strings.Contains(prompt, "Test Seed") {
		t.Error("prompt should contain the seed title")
	}
	if !strings.Contains(prompt, "A description of the seed topic.") {
		t.Error("prompt should contain the seed description")
	}
	if !strings.Contains(prompt, "**Description:**") {
		t.Error("prompt should include the Description label when description is present")
	}
}

func TestBuildSynthesisLLMPrompt_WithoutDescription(t *testing.T) {
	seed := SeedInfo{
		Title:       "No Desc Seed",
		Description: "",
	}
	leaves := defaultSourceLeaves()

	prompt := buildSynthesisLLMPrompt(seed, leaves)

	if !strings.Contains(prompt, "No Desc Seed") {
		t.Error("prompt should contain the seed title")
	}
	if strings.Contains(prompt, "**Description:**") {
		t.Error("prompt should NOT include the Description label when description is empty")
	}
}

func TestBuildSynthesisLLMPrompt_MultipleSourceLeaves(t *testing.T) {
	seed := SeedInfo{Title: "Seed"}
	leaves := defaultSourceLeaves()

	prompt := buildSynthesisLLMPrompt(seed, leaves)

	if !strings.Contains(prompt, "Idea 1: First Idea") {
		t.Error("prompt should contain first leaf title with numbered heading")
	}
	if !strings.Contains(prompt, "Idea 2: Second Idea") {
		t.Error("prompt should contain second leaf title with numbered heading")
	}
	if !strings.Contains(prompt, "Summary of the first idea.") {
		t.Error("prompt should contain first leaf summary")
	}
	if !strings.Contains(prompt, "Summary of the second idea.") {
		t.Error("prompt should contain second leaf summary")
	}
}

func TestBuildSynthesisLLMPrompt_KeyPointsIncluded(t *testing.T) {
	seed := SeedInfo{Title: "Seed"}
	leaves := defaultSourceLeaves()

	prompt := buildSynthesisLLMPrompt(seed, leaves)

	if !strings.Contains(prompt, "**Key Points:**") {
		t.Error("prompt should include Key Points section")
	}
	if !strings.Contains(prompt, "- KP1") {
		t.Error("prompt should contain key point KP1")
	}
	if !strings.Contains(prompt, "- KP2") {
		t.Error("prompt should contain key point KP2")
	}
	if !strings.Contains(prompt, "- KP3") {
		t.Error("prompt should contain key point KP3")
	}
}

func TestBuildSynthesisLLMPrompt_OpenQuestionsIncluded(t *testing.T) {
	seed := SeedInfo{Title: "Seed"}
	leaves := defaultSourceLeaves()

	prompt := buildSynthesisLLMPrompt(seed, leaves)

	if !strings.Contains(prompt, "**Open Questions:**") {
		t.Error("prompt should include Open Questions section")
	}
	if !strings.Contains(prompt, "- OQ1") {
		t.Error("prompt should contain open question OQ1")
	}
	if !strings.Contains(prompt, "- OQ2") {
		t.Error("prompt should contain open question OQ2")
	}
}

func TestBuildSynthesisLLMPrompt_NoKeyPointsOrQuestions(t *testing.T) {
	seed := SeedInfo{Title: "Seed"}
	leaves := []SourceLeaf{
		{
			ID:      types.LeafIDFrom("leaf_1"),
			SeedID:  types.SeedIDFrom("seed_test"),
			Title:   "Bare Leaf",
			Summary: "A leaf with no key points or questions.",
		},
		{
			ID:      types.LeafIDFrom("leaf_2"),
			SeedID:  types.SeedIDFrom("seed_test"),
			Title:   "Another Bare Leaf",
			Summary: "Also minimal.",
		},
	}

	prompt := buildSynthesisLLMPrompt(seed, leaves)

	if strings.Contains(prompt, "**Key Points:**") {
		t.Error("prompt should NOT include Key Points section when leaves have none")
	}
	if strings.Contains(prompt, "**Open Questions:**") {
		t.Error("prompt should NOT include Open Questions section when leaves have none")
	}
}

func TestBuildSynthesisLLMPrompt_ContainsJSONInstructions(t *testing.T) {
	seed := SeedInfo{Title: "Seed"}
	leaves := defaultSourceLeaves()

	prompt := buildSynthesisLLMPrompt(seed, leaves)

	if !strings.Contains(prompt, "JSON object") {
		t.Error("prompt should instruct LLM to respond with JSON")
	}
	if !strings.Contains(prompt, `"title"`) {
		t.Error("prompt should show the expected JSON schema with title field")
	}
	if !strings.Contains(prompt, `"summary"`) {
		t.Error("prompt should show the expected JSON schema with summary field")
	}
	if !strings.Contains(prompt, `"key_points"`) {
		t.Error("prompt should show the expected JSON schema with key_points field")
	}
}

// --- StartSynthesis full flow tests ---

func TestStartSynthesis_Success(t *testing.T) {
	repo := &stubSynthesisRepo{}
	llmProv := &stubLLMProvider{
		chatCompletionFn: func(_ context.Context, _ llm.ChatRequest) (llm.ChatResponse, error) {
			return llm.ChatResponse{Content: validLLMJSON()}, nil
		},
	}
	leafReader := &stubSourceLeafReader{
		findByIDsFn: func(_ context.Context, ids []types.LeafID) ([]SourceLeaf, error) {
			return defaultSourceLeaves(), nil
		},
	}
	seedReader := &stubSeedReader{
		findByIDFn: func(_ context.Context, _ types.SeedID) (SeedInfo, error) {
			return SeedInfo{Title: "Test Seed", Description: "Desc"}, nil
		},
	}
	leafCreator := &stubLeafCreator{
		createFn: func(_ context.Context, _ SynthesisLeafParams) (types.LeafID, error) {
			return types.LeafIDFrom("leaf_result"), nil
		},
	}
	wsMembers := &stubWSMembers{}
	pub := &stubPublisher{}

	svc := newTestService(repo, llmProv, leafReader, seedReader, leafCreator, wsMembers, pub)

	ctx := testCtx("user_test")
	wsID := types.WorkspaceIDFrom("ws_test")
	sourceIDs := []types.LeafID{
		types.LeafIDFrom("leaf_1"),
		types.LeafIDFrom("leaf_2"),
	}

	workflow, err := svc.StartSynthesis(ctx, wsID, sourceIDs)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Workflow should be completed.
	if workflow.Status() != domain.WorkflowCompleted {
		t.Errorf("expected status %s, got %s", domain.WorkflowCompleted, workflow.Status())
	}

	// Result leaf should be set.
	if workflow.ResultLeafID().String() != "leaf_result" {
		t.Errorf("expected result leaf ID 'leaf_result', got %q", workflow.ResultLeafID().String())
	}

	// Repo should have been called: Create + Update (for completion).
	if repo.updatedWorkflow.Status() != domain.WorkflowCompleted {
		t.Errorf("expected updated workflow to be completed, got %s", repo.updatedWorkflow.Status())
	}

	// Leaf creator should have been called once.
	if leafCreator.createCalls != 1 {
		t.Errorf("expected 1 leaf creator call, got %d", leafCreator.createCalls)
	}

	// Verify leaf creation params.
	params := leafCreator.lastParams
	if params.Title != "Synthesized Insight" {
		t.Errorf("expected leaf title 'Synthesized Insight', got %q", params.Title)
	}
	if params.WorkspaceID.String() != "ws_test" {
		t.Errorf("expected workspace ID 'ws_test', got %q", params.WorkspaceID.String())
	}
	if params.AuthorID.String() != "user_test" {
		t.Errorf("expected author ID 'user_test', got %q", params.AuthorID.String())
	}
	if len(params.Sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(params.Sources))
	}

	// Events: started + completed + leaf_created = 3
	if pub.publishCalls != 3 {
		t.Errorf("expected 3 publish calls, got %d", pub.publishCalls)
	}
}

func TestStartSynthesis_LLMFailure_WorkflowMarkedFailed(t *testing.T) {
	repo := &stubSynthesisRepo{}
	llmProv := &stubLLMProvider{
		chatCompletionFn: func(_ context.Context, _ llm.ChatRequest) (llm.ChatResponse, error) {
			return llm.ChatResponse{}, errors.New("llm service unavailable")
		},
	}
	leafReader := &stubSourceLeafReader{
		findByIDsFn: func(_ context.Context, _ []types.LeafID) ([]SourceLeaf, error) {
			return defaultSourceLeaves(), nil
		},
	}
	seedReader := &stubSeedReader{
		findByIDFn: func(_ context.Context, _ types.SeedID) (SeedInfo, error) {
			return SeedInfo{Title: "Seed"}, nil
		},
	}
	leafCreator := &stubLeafCreator{}
	wsMembers := &stubWSMembers{}
	pub := &stubPublisher{}

	svc := newTestService(repo, llmProv, leafReader, seedReader, leafCreator, wsMembers, pub)

	ctx := testCtx("user_test")
	wsID := types.WorkspaceIDFrom("ws_test")
	sourceIDs := []types.LeafID{
		types.LeafIDFrom("leaf_1"),
		types.LeafIDFrom("leaf_2"),
	}

	workflow, err := svc.StartSynthesis(ctx, wsID, sourceIDs)
	if err != nil {
		t.Fatalf("expected no hard error (graceful failure), got: %v", err)
	}

	// Workflow should be failed, not errored out.
	if workflow.Status() != domain.WorkflowFailed {
		t.Errorf("expected status %s, got %s", domain.WorkflowFailed, workflow.Status())
	}

	if !strings.Contains(workflow.FailureReason(), "llm call") {
		t.Errorf("expected failure reason to mention 'llm call', got %q", workflow.FailureReason())
	}

	// Leaf creator should NOT have been called.
	if leafCreator.createCalls != 0 {
		t.Errorf("expected 0 leaf creator calls, got %d", leafCreator.createCalls)
	}

	// Repo should have Update called (to persist failed state).
	if repo.updateCalls != 1 {
		t.Errorf("expected 1 update call (for failure), got %d", repo.updateCalls)
	}
}

func TestStartSynthesis_LeafCreationFailure_WorkflowMarkedFailed(t *testing.T) {
	repo := &stubSynthesisRepo{}
	llmProv := &stubLLMProvider{
		chatCompletionFn: func(_ context.Context, _ llm.ChatRequest) (llm.ChatResponse, error) {
			return llm.ChatResponse{Content: validLLMJSON()}, nil
		},
	}
	leafReader := &stubSourceLeafReader{
		findByIDsFn: func(_ context.Context, _ []types.LeafID) ([]SourceLeaf, error) {
			return defaultSourceLeaves(), nil
		},
	}
	seedReader := &stubSeedReader{
		findByIDFn: func(_ context.Context, _ types.SeedID) (SeedInfo, error) {
			return SeedInfo{Title: "Seed"}, nil
		},
	}
	leafCreator := &stubLeafCreator{
		createFn: func(_ context.Context, _ SynthesisLeafParams) (types.LeafID, error) {
			return types.LeafID{}, errors.New("database write failed")
		},
	}
	wsMembers := &stubWSMembers{}
	pub := &stubPublisher{}

	svc := newTestService(repo, llmProv, leafReader, seedReader, leafCreator, wsMembers, pub)

	ctx := testCtx("user_test")
	wsID := types.WorkspaceIDFrom("ws_test")
	sourceIDs := []types.LeafID{
		types.LeafIDFrom("leaf_1"),
		types.LeafIDFrom("leaf_2"),
	}

	workflow, err := svc.StartSynthesis(ctx, wsID, sourceIDs)
	if err != nil {
		t.Fatalf("expected no hard error (graceful failure), got: %v", err)
	}

	if workflow.Status() != domain.WorkflowFailed {
		t.Errorf("expected status %s, got %s", domain.WorkflowFailed, workflow.Status())
	}

	if !strings.Contains(workflow.FailureReason(), "create synthesis leaf") {
		t.Errorf("expected failure reason to mention 'create synthesis leaf', got %q", workflow.FailureReason())
	}
}

func TestStartSynthesis_SourceLeavesNotFound_WorkflowMarkedFailed(t *testing.T) {
	repo := &stubSynthesisRepo{}
	llmProv := &stubLLMProvider{}
	leafReader := &stubSourceLeafReader{
		findByIDsFn: func(_ context.Context, ids []types.LeafID) ([]SourceLeaf, error) {
			// Return fewer leaves than requested — simulates missing leaves.
			return []SourceLeaf{defaultSourceLeaves()[0]}, nil
		},
	}
	seedReader := &stubSeedReader{
		findByIDFn: func(_ context.Context, _ types.SeedID) (SeedInfo, error) {
			return SeedInfo{Title: "Seed"}, nil
		},
	}
	leafCreator := &stubLeafCreator{}
	wsMembers := &stubWSMembers{}
	pub := &stubPublisher{}

	svc := newTestService(repo, llmProv, leafReader, seedReader, leafCreator, wsMembers, pub)

	ctx := testCtx("user_test")
	wsID := types.WorkspaceIDFrom("ws_test")
	sourceIDs := []types.LeafID{
		types.LeafIDFrom("leaf_1"),
		types.LeafIDFrom("leaf_2"),
	}

	workflow, err := svc.StartSynthesis(ctx, wsID, sourceIDs)
	if err != nil {
		t.Fatalf("expected no hard error (graceful failure), got: %v", err)
	}

	if workflow.Status() != domain.WorkflowFailed {
		t.Errorf("expected status %s, got %s", domain.WorkflowFailed, workflow.Status())
	}

	if !strings.Contains(workflow.FailureReason(), "expected 2 source leaves, found 1") {
		t.Errorf("expected failure reason about leaf count mismatch, got %q", workflow.FailureReason())
	}

	// LLM should NOT have been called.
	if llmProv.lastRequest.Messages != nil {
		t.Error("expected no LLM call when source leaves are missing")
	}
}

func TestStartSynthesis_SourceLeafReadError_WorkflowMarkedFailed(t *testing.T) {
	repo := &stubSynthesisRepo{}
	llmProv := &stubLLMProvider{}
	leafReader := &stubSourceLeafReader{
		findByIDsFn: func(_ context.Context, _ []types.LeafID) ([]SourceLeaf, error) {
			return nil, errors.New("database connection lost")
		},
	}
	seedReader := &stubSeedReader{}
	leafCreator := &stubLeafCreator{}
	wsMembers := &stubWSMembers{}
	pub := &stubPublisher{}

	svc := newTestService(repo, llmProv, leafReader, seedReader, leafCreator, wsMembers, pub)

	ctx := testCtx("user_test")
	wsID := types.WorkspaceIDFrom("ws_test")
	sourceIDs := []types.LeafID{
		types.LeafIDFrom("leaf_1"),
		types.LeafIDFrom("leaf_2"),
	}

	workflow, err := svc.StartSynthesis(ctx, wsID, sourceIDs)
	if err != nil {
		t.Fatalf("expected no hard error (graceful failure), got: %v", err)
	}

	if workflow.Status() != domain.WorkflowFailed {
		t.Errorf("expected status %s, got %s", domain.WorkflowFailed, workflow.Status())
	}

	if !strings.Contains(workflow.FailureReason(), "load source leaves") {
		t.Errorf("expected failure reason about loading source leaves, got %q", workflow.FailureReason())
	}
}

func TestStartSynthesis_SeedReadError_WorkflowMarkedFailed(t *testing.T) {
	repo := &stubSynthesisRepo{}
	llmProv := &stubLLMProvider{}
	leafReader := &stubSourceLeafReader{
		findByIDsFn: func(_ context.Context, _ []types.LeafID) ([]SourceLeaf, error) {
			return defaultSourceLeaves(), nil
		},
	}
	seedReader := &stubSeedReader{
		findByIDFn: func(_ context.Context, _ types.SeedID) (SeedInfo, error) {
			return SeedInfo{}, errors.New("seed not found")
		},
	}
	leafCreator := &stubLeafCreator{}
	wsMembers := &stubWSMembers{}
	pub := &stubPublisher{}

	svc := newTestService(repo, llmProv, leafReader, seedReader, leafCreator, wsMembers, pub)

	ctx := testCtx("user_test")
	wsID := types.WorkspaceIDFrom("ws_test")
	sourceIDs := []types.LeafID{
		types.LeafIDFrom("leaf_1"),
		types.LeafIDFrom("leaf_2"),
	}

	workflow, err := svc.StartSynthesis(ctx, wsID, sourceIDs)
	if err != nil {
		t.Fatalf("expected no hard error (graceful failure), got: %v", err)
	}

	if workflow.Status() != domain.WorkflowFailed {
		t.Errorf("expected status %s, got %s", domain.WorkflowFailed, workflow.Status())
	}

	if !strings.Contains(workflow.FailureReason(), "load seed") {
		t.Errorf("expected failure reason about loading seed, got %q", workflow.FailureReason())
	}
}

func TestStartSynthesis_LLMReturnsInvalidJSON_WorkflowMarkedFailed(t *testing.T) {
	repo := &stubSynthesisRepo{}
	llmProv := &stubLLMProvider{
		chatCompletionFn: func(_ context.Context, _ llm.ChatRequest) (llm.ChatResponse, error) {
			return llm.ChatResponse{Content: "I'm sorry, I can't do that."}, nil
		},
	}
	leafReader := &stubSourceLeafReader{
		findByIDsFn: func(_ context.Context, _ []types.LeafID) ([]SourceLeaf, error) {
			return defaultSourceLeaves(), nil
		},
	}
	seedReader := &stubSeedReader{
		findByIDFn: func(_ context.Context, _ types.SeedID) (SeedInfo, error) {
			return SeedInfo{Title: "Seed"}, nil
		},
	}
	leafCreator := &stubLeafCreator{}
	wsMembers := &stubWSMembers{}
	pub := &stubPublisher{}

	svc := newTestService(repo, llmProv, leafReader, seedReader, leafCreator, wsMembers, pub)

	ctx := testCtx("user_test")
	wsID := types.WorkspaceIDFrom("ws_test")
	sourceIDs := []types.LeafID{
		types.LeafIDFrom("leaf_1"),
		types.LeafIDFrom("leaf_2"),
	}

	workflow, err := svc.StartSynthesis(ctx, wsID, sourceIDs)
	if err != nil {
		t.Fatalf("expected no hard error (graceful failure), got: %v", err)
	}

	if workflow.Status() != domain.WorkflowFailed {
		t.Errorf("expected status %s, got %s", domain.WorkflowFailed, workflow.Status())
	}

	if !strings.Contains(workflow.FailureReason(), "parse llm response") {
		t.Errorf("expected failure reason about parsing LLM response, got %q", workflow.FailureReason())
	}
}

func TestStartSynthesis_NoAuthClaims_ReturnsError(t *testing.T) {
	repo := &stubSynthesisRepo{}
	llmProv := &stubLLMProvider{}
	leafReader := &stubSourceLeafReader{}
	seedReader := &stubSeedReader{}
	leafCreator := &stubLeafCreator{}
	wsMembers := &stubWSMembers{}
	pub := &stubPublisher{}

	svc := newTestService(repo, llmProv, leafReader, seedReader, leafCreator, wsMembers, pub)

	ctx := testCtxNoAuth()
	wsID := types.WorkspaceIDFrom("ws_test")
	sourceIDs := []types.LeafID{
		types.LeafIDFrom("leaf_1"),
		types.LeafIDFrom("leaf_2"),
	}

	_, err := svc.StartSynthesis(ctx, wsID, sourceIDs)
	if err == nil {
		t.Fatal("expected error when no auth claims in context")
	}

	if !canopyerr.Is(err, canopyerr.ErrUnauthenticated) {
		t.Errorf("expected unauthenticated error, got: %v", err)
	}
}

func TestStartSynthesis_NonMember_ReturnsUnauthorized(t *testing.T) {
	repo := &stubSynthesisRepo{}
	llmProv := &stubLLMProvider{}
	leafReader := &stubSourceLeafReader{}
	seedReader := &stubSeedReader{}
	leafCreator := &stubLeafCreator{}
	wsMembers := &stubWSMembers{
		findMemberFn: func(_ context.Context, _ types.WorkspaceID, _ types.UserID) (wsdomain.WorkspaceMember, error) {
			return wsdomain.WorkspaceMember{}, canopyerr.ErrNotFound
		},
	}
	pub := &stubPublisher{}

	svc := newTestService(repo, llmProv, leafReader, seedReader, leafCreator, wsMembers, pub)

	ctx := testCtx("user_outsider")
	wsID := types.WorkspaceIDFrom("ws_test")
	sourceIDs := []types.LeafID{
		types.LeafIDFrom("leaf_1"),
		types.LeafIDFrom("leaf_2"),
	}

	_, err := svc.StartSynthesis(ctx, wsID, sourceIDs)
	if err == nil {
		t.Fatal("expected error when user is not a workspace member")
	}

	if !canopyerr.Is(err, canopyerr.ErrUnauthorized) {
		t.Errorf("expected unauthorized error, got: %v", err)
	}
}

func TestStartSynthesis_TooFewLeaves_ReturnsError(t *testing.T) {
	repo := &stubSynthesisRepo{}
	llmProv := &stubLLMProvider{}
	leafReader := &stubSourceLeafReader{}
	seedReader := &stubSeedReader{}
	leafCreator := &stubLeafCreator{}
	wsMembers := &stubWSMembers{}
	pub := &stubPublisher{}

	svc := newTestService(repo, llmProv, leafReader, seedReader, leafCreator, wsMembers, pub)

	ctx := testCtx("user_test")
	wsID := types.WorkspaceIDFrom("ws_test")
	sourceIDs := []types.LeafID{
		types.LeafIDFrom("leaf_1"),
	}

	_, err := svc.StartSynthesis(ctx, wsID, sourceIDs)
	if err == nil {
		t.Fatal("expected error when fewer than 2 source leaves provided")
	}

	if !strings.Contains(err.Error(), "at least 2 source leaves") {
		t.Errorf("expected error about minimum leaf count, got: %v", err)
	}
}

func TestStartSynthesis_RepoCreateFailure_ReturnsError(t *testing.T) {
	repo := &stubSynthesisRepo{
		createFn: func(_ context.Context, _ domain.SynthesisWorkflow) error {
			return errors.New("database error")
		},
	}
	llmProv := &stubLLMProvider{}
	leafReader := &stubSourceLeafReader{}
	seedReader := &stubSeedReader{}
	leafCreator := &stubLeafCreator{}
	wsMembers := &stubWSMembers{}
	pub := &stubPublisher{}

	svc := newTestService(repo, llmProv, leafReader, seedReader, leafCreator, wsMembers, pub)

	ctx := testCtx("user_test")
	wsID := types.WorkspaceIDFrom("ws_test")
	sourceIDs := []types.LeafID{
		types.LeafIDFrom("leaf_1"),
		types.LeafIDFrom("leaf_2"),
	}

	_, err := svc.StartSynthesis(ctx, wsID, sourceIDs)
	if err == nil {
		t.Fatal("expected error when repo.Create fails")
	}
}

func TestStartSynthesis_PublishesStartedEvent(t *testing.T) {
	repo := &stubSynthesisRepo{}
	llmProv := &stubLLMProvider{
		chatCompletionFn: func(_ context.Context, _ llm.ChatRequest) (llm.ChatResponse, error) {
			return llm.ChatResponse{Content: validLLMJSON()}, nil
		},
	}
	leafReader := &stubSourceLeafReader{
		findByIDsFn: func(_ context.Context, _ []types.LeafID) ([]SourceLeaf, error) {
			return defaultSourceLeaves(), nil
		},
	}
	seedReader := &stubSeedReader{
		findByIDFn: func(_ context.Context, _ types.SeedID) (SeedInfo, error) {
			return SeedInfo{Title: "Seed"}, nil
		},
	}
	leafCreator := &stubLeafCreator{}
	wsMembers := &stubWSMembers{}
	pub := &stubPublisher{}

	svc := newTestService(repo, llmProv, leafReader, seedReader, leafCreator, wsMembers, pub)

	ctx := testCtx("user_test")
	wsID := types.WorkspaceIDFrom("ws_test")
	sourceIDs := []types.LeafID{
		types.LeafIDFrom("leaf_1"),
		types.LeafIDFrom("leaf_2"),
	}

	_, err := svc.StartSynthesis(ctx, wsID, sourceIDs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the started event was published first.
	if len(pub.published) < 1 {
		t.Fatal("expected at least 1 published event")
	}

	startedEvent := pub.published[0]
	if startedEvent.Type != domain.SubjectSynthesisStarted {
		t.Errorf("expected first event type %q, got %q", domain.SubjectSynthesisStarted, startedEvent.Type)
	}
	if startedEvent.WorkspaceID != "ws_test" {
		t.Errorf("expected workspace ID 'ws_test' in event, got %q", startedEvent.WorkspaceID)
	}

	// Verify the subject follows the convention.
	expectedSubjectPrefix := "workspace.ws_test.synthesis."
	if !strings.HasPrefix(startedEvent.Subject, expectedSubjectPrefix) {
		t.Errorf("expected subject to start with %q, got %q", expectedSubjectPrefix, startedEvent.Subject)
	}
}

func TestStartSynthesis_SuccessPublishesThreeEvents(t *testing.T) {
	repo := &stubSynthesisRepo{}
	llmProv := &stubLLMProvider{
		chatCompletionFn: func(_ context.Context, _ llm.ChatRequest) (llm.ChatResponse, error) {
			return llm.ChatResponse{Content: validLLMJSON()}, nil
		},
	}
	leafReader := &stubSourceLeafReader{
		findByIDsFn: func(_ context.Context, _ []types.LeafID) ([]SourceLeaf, error) {
			return defaultSourceLeaves(), nil
		},
	}
	seedReader := &stubSeedReader{
		findByIDFn: func(_ context.Context, _ types.SeedID) (SeedInfo, error) {
			return SeedInfo{Title: "Seed"}, nil
		},
	}
	leafCreator := &stubLeafCreator{}
	wsMembers := &stubWSMembers{}
	pub := &stubPublisher{}

	svc := newTestService(repo, llmProv, leafReader, seedReader, leafCreator, wsMembers, pub)

	ctx := testCtx("user_test")
	wsID := types.WorkspaceIDFrom("ws_test")
	sourceIDs := []types.LeafID{
		types.LeafIDFrom("leaf_1"),
		types.LeafIDFrom("leaf_2"),
	}

	_, err := svc.StartSynthesis(ctx, wsID, sourceIDs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pub.published) != 3 {
		t.Fatalf("expected 3 events, got %d", len(pub.published))
	}

	// Event order: started, completed, leaf_created.
	expectedTypes := []string{
		domain.SubjectSynthesisStarted,
		domain.SubjectSynthesisCompleted,
		domain.SubjectSynthesisLeafCreated,
	}
	for i, expected := range expectedTypes {
		if pub.published[i].Type != expected {
			t.Errorf("event[%d]: expected type %q, got %q", i, expected, pub.published[i].Type)
		}
	}
}

func TestStartSynthesis_LLMCalledWithCorrectPrompt(t *testing.T) {
	repo := &stubSynthesisRepo{}
	llmProv := &stubLLMProvider{
		chatCompletionFn: func(_ context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
			return llm.ChatResponse{Content: validLLMJSON()}, nil
		},
	}
	leafReader := &stubSourceLeafReader{
		findByIDsFn: func(_ context.Context, _ []types.LeafID) ([]SourceLeaf, error) {
			return defaultSourceLeaves(), nil
		},
	}
	seedReader := &stubSeedReader{
		findByIDFn: func(_ context.Context, _ types.SeedID) (SeedInfo, error) {
			return SeedInfo{Title: "Test Seed", Description: "Test Description"}, nil
		},
	}
	leafCreator := &stubLeafCreator{}
	wsMembers := &stubWSMembers{}
	pub := &stubPublisher{}

	svc := newTestService(repo, llmProv, leafReader, seedReader, leafCreator, wsMembers, pub)

	ctx := testCtx("user_test")
	wsID := types.WorkspaceIDFrom("ws_test")
	sourceIDs := []types.LeafID{
		types.LeafIDFrom("leaf_1"),
		types.LeafIDFrom("leaf_2"),
	}

	_, err := svc.StartSynthesis(ctx, wsID, sourceIDs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify LLM received 2 messages: system prompt + user prompt.
	if len(llmProv.lastRequest.Messages) != 2 {
		t.Fatalf("expected 2 messages to LLM, got %d", len(llmProv.lastRequest.Messages))
	}

	systemMsg := llmProv.lastRequest.Messages[0]
	if systemMsg.Role != "system" {
		t.Errorf("expected first message role 'system', got %q", systemMsg.Role)
	}
	if !strings.Contains(systemMsg.Content, "Test Seed") {
		t.Error("system prompt should contain the seed title")
	}
	if !strings.Contains(systemMsg.Content, "Test Description") {
		t.Error("system prompt should contain the seed description")
	}
	if !strings.Contains(systemMsg.Content, "First Idea") {
		t.Error("system prompt should contain source leaf titles")
	}

	userMsg := llmProv.lastRequest.Messages[1]
	if userMsg.Role != "user" {
		t.Errorf("expected second message role 'user', got %q", userMsg.Role)
	}
	if !strings.Contains(userMsg.Content, "synthesize") {
		t.Error("user prompt should ask to synthesize")
	}
}

// --- FindByWorkspace tests ---

func TestFindByWorkspace_Success(t *testing.T) {
	wsID := types.WorkspaceIDFrom("ws_test")
	expected := []domain.SynthesisWorkflow{}

	repo := &stubSynthesisRepo{
		findByWSFn: func(_ context.Context, id types.WorkspaceID) ([]domain.SynthesisWorkflow, error) {
			if id.String() != wsID.String() {
				t.Errorf("expected workspace ID %q, got %q", wsID.String(), id.String())
			}
			return expected, nil
		},
	}
	wsMembers := &stubWSMembers{}
	pub := &stubPublisher{}

	svc := newTestService(repo, &stubLLMProvider{}, &stubSourceLeafReader{}, &stubSeedReader{}, &stubLeafCreator{}, wsMembers, pub)

	ctx := testCtx("user_test")
	workflows, err := svc.FindByWorkspace(ctx, wsID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if workflows == nil {
		t.Error("expected non-nil workflows slice")
	}
}

func TestFindByWorkspace_NoAuth_ReturnsError(t *testing.T) {
	repo := &stubSynthesisRepo{}
	wsMembers := &stubWSMembers{}
	pub := &stubPublisher{}

	svc := newTestService(repo, &stubLLMProvider{}, &stubSourceLeafReader{}, &stubSeedReader{}, &stubLeafCreator{}, wsMembers, pub)

	ctx := testCtxNoAuth()
	wsID := types.WorkspaceIDFrom("ws_test")

	_, err := svc.FindByWorkspace(ctx, wsID)
	if err == nil {
		t.Fatal("expected error when no auth claims")
	}
}

func TestFindByWorkspace_NonMember_ReturnsUnauthorized(t *testing.T) {
	repo := &stubSynthesisRepo{}
	wsMembers := &stubWSMembers{
		findMemberFn: func(_ context.Context, _ types.WorkspaceID, _ types.UserID) (wsdomain.WorkspaceMember, error) {
			return wsdomain.WorkspaceMember{}, canopyerr.ErrNotFound
		},
	}
	pub := &stubPublisher{}

	svc := newTestService(repo, &stubLLMProvider{}, &stubSourceLeafReader{}, &stubSeedReader{}, &stubLeafCreator{}, wsMembers, pub)

	ctx := testCtx("user_outsider")
	wsID := types.WorkspaceIDFrom("ws_test")

	_, err := svc.FindByWorkspace(ctx, wsID)
	if err == nil {
		t.Fatal("expected error for non-member")
	}
	if !canopyerr.Is(err, canopyerr.ErrUnauthorized) {
		t.Errorf("expected unauthorized error, got: %v", err)
	}
}

// --- FailSynthesis tests ---

func TestFailSynthesis_Success(t *testing.T) {
	// Create a processing workflow that can be failed.
	wsID := types.WorkspaceIDFrom("ws_test")
	userID := types.UserIDFrom("user_test")
	leafIDs := []types.LeafID{types.LeafIDFrom("leaf_1"), types.LeafIDFrom("leaf_2")}
	workflow, err := domain.NewSynthesisWorkflow(wsID, userID, leafIDs)
	if err != nil {
		t.Fatalf("failed to create workflow: %v", err)
	}
	if err := workflow.Start(); err != nil {
		t.Fatalf("failed to start workflow: %v", err)
	}

	repo := &stubSynthesisRepo{
		findByIDFn: func(_ context.Context, _ domain.SynthesisID) (domain.SynthesisWorkflow, error) {
			return workflow, nil
		},
	}
	pub := &stubPublisher{}

	svc := newTestService(repo, &stubLLMProvider{}, &stubSourceLeafReader{}, &stubSeedReader{}, &stubLeafCreator{}, &stubWSMembers{}, pub)

	err = svc.FailSynthesis(context.Background(), workflow.ID(), "manual failure")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.updatedWorkflow.Status() != domain.WorkflowFailed {
		t.Errorf("expected updated workflow status %s, got %s", domain.WorkflowFailed, repo.updatedWorkflow.Status())
	}
}

func TestFailSynthesis_NotFound_ReturnsError(t *testing.T) {
	repo := &stubSynthesisRepo{
		findByIDFn: func(_ context.Context, _ domain.SynthesisID) (domain.SynthesisWorkflow, error) {
			return domain.SynthesisWorkflow{}, canopyerr.ErrNotFound
		},
	}
	pub := &stubPublisher{}

	svc := newTestService(repo, &stubLLMProvider{}, &stubSourceLeafReader{}, &stubSeedReader{}, &stubLeafCreator{}, &stubWSMembers{}, pub)

	err := svc.FailSynthesis(context.Background(), domain.SynthesisIDFrom("syn_nonexistent"), "reason")
	if err == nil {
		t.Fatal("expected error when workflow not found")
	}
}

// --- CompleteSynthesis tests ---

func TestCompleteSynthesis_Success(t *testing.T) {
	wsID := types.WorkspaceIDFrom("ws_test")
	userID := types.UserIDFrom("user_test")
	leafIDs := []types.LeafID{types.LeafIDFrom("leaf_1"), types.LeafIDFrom("leaf_2")}
	workflow, err := domain.NewSynthesisWorkflow(wsID, userID, leafIDs)
	if err != nil {
		t.Fatalf("failed to create workflow: %v", err)
	}
	if err := workflow.Start(); err != nil {
		t.Fatalf("failed to start workflow: %v", err)
	}

	repo := &stubSynthesisRepo{
		findByIDFn: func(_ context.Context, _ domain.SynthesisID) (domain.SynthesisWorkflow, error) {
			return workflow, nil
		},
	}
	pub := &stubPublisher{}

	svc := newTestService(repo, &stubLLMProvider{}, &stubSourceLeafReader{}, &stubSeedReader{}, &stubLeafCreator{}, &stubWSMembers{}, pub)

	resultLeafID := types.LeafIDFrom("leaf_result")
	completed, err := svc.CompleteSynthesis(context.Background(), workflow.ID(), resultLeafID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if completed.Status() != domain.WorkflowCompleted {
		t.Errorf("expected status %s, got %s", domain.WorkflowCompleted, completed.Status())
	}
	if completed.ResultLeafID().String() != "leaf_result" {
		t.Errorf("expected result leaf ID 'leaf_result', got %q", completed.ResultLeafID().String())
	}

	// Should publish completed event.
	if pub.publishCalls != 1 {
		t.Errorf("expected 1 publish call, got %d", pub.publishCalls)
	}
	if pub.published[0].Type != domain.SubjectSynthesisCompleted {
		t.Errorf("expected event type %q, got %q", domain.SubjectSynthesisCompleted, pub.published[0].Type)
	}
}

func TestCompleteSynthesis_NotProcessing_ReturnsError(t *testing.T) {
	// Workflow in pending state cannot be completed directly.
	wsID := types.WorkspaceIDFrom("ws_test")
	userID := types.UserIDFrom("user_test")
	leafIDs := []types.LeafID{types.LeafIDFrom("leaf_1"), types.LeafIDFrom("leaf_2")}
	workflow, err := domain.NewSynthesisWorkflow(wsID, userID, leafIDs)
	if err != nil {
		t.Fatalf("failed to create workflow: %v", err)
	}
	// Do NOT call Start() — workflow is still pending.

	repo := &stubSynthesisRepo{
		findByIDFn: func(_ context.Context, _ domain.SynthesisID) (domain.SynthesisWorkflow, error) {
			return workflow, nil
		},
	}
	pub := &stubPublisher{}

	svc := newTestService(repo, &stubLLMProvider{}, &stubSourceLeafReader{}, &stubSeedReader{}, &stubLeafCreator{}, &stubWSMembers{}, pub)

	_, err = svc.CompleteSynthesis(context.Background(), workflow.ID(), types.LeafIDFrom("leaf_result"))
	if err == nil {
		t.Fatal("expected error when completing a non-processing workflow")
	}
}

func TestCompleteSynthesis_ZeroResultLeafID_ReturnsError(t *testing.T) {
	wsID := types.WorkspaceIDFrom("ws_test")
	userID := types.UserIDFrom("user_test")
	leafIDs := []types.LeafID{types.LeafIDFrom("leaf_1"), types.LeafIDFrom("leaf_2")}
	workflow, err := domain.NewSynthesisWorkflow(wsID, userID, leafIDs)
	if err != nil {
		t.Fatalf("failed to create workflow: %v", err)
	}
	if err := workflow.Start(); err != nil {
		t.Fatalf("failed to start workflow: %v", err)
	}

	repo := &stubSynthesisRepo{
		findByIDFn: func(_ context.Context, _ domain.SynthesisID) (domain.SynthesisWorkflow, error) {
			return workflow, nil
		},
	}
	pub := &stubPublisher{}

	svc := newTestService(repo, &stubLLMProvider{}, &stubSourceLeafReader{}, &stubSeedReader{}, &stubLeafCreator{}, &stubWSMembers{}, pub)

	// Pass a zero-value LeafID.
	_, err = svc.CompleteSynthesis(context.Background(), workflow.ID(), types.LeafID{})
	if err == nil {
		t.Fatal("expected error when result leaf ID is zero")
	}
}

// --- Source attribution tests ---

func TestStartSynthesis_SourceAttributionPassedToLeafCreator(t *testing.T) {
	repo := &stubSynthesisRepo{}
	llmProv := &stubLLMProvider{
		chatCompletionFn: func(_ context.Context, _ llm.ChatRequest) (llm.ChatResponse, error) {
			return llm.ChatResponse{Content: validLLMJSON()}, nil
		},
	}
	leafReader := &stubSourceLeafReader{
		findByIDsFn: func(_ context.Context, _ []types.LeafID) ([]SourceLeaf, error) {
			return defaultSourceLeaves(), nil
		},
	}
	seedReader := &stubSeedReader{
		findByIDFn: func(_ context.Context, _ types.SeedID) (SeedInfo, error) {
			return SeedInfo{Title: "Seed"}, nil
		},
	}
	leafCreator := &stubLeafCreator{}
	wsMembers := &stubWSMembers{}
	pub := &stubPublisher{}

	svc := newTestService(repo, llmProv, leafReader, seedReader, leafCreator, wsMembers, pub)

	ctx := testCtx("user_test")
	wsID := types.WorkspaceIDFrom("ws_test")
	sourceIDs := []types.LeafID{
		types.LeafIDFrom("leaf_1"),
		types.LeafIDFrom("leaf_2"),
	}

	_, err := svc.StartSynthesis(ctx, wsID, sourceIDs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	params := leafCreator.lastParams
	if len(params.Sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(params.Sources))
	}

	if params.Sources[0].LeafID != "leaf_1" {
		t.Errorf("expected source[0] leaf ID 'leaf_1', got %q", params.Sources[0].LeafID)
	}
	if params.Sources[0].Title != "First Idea" {
		t.Errorf("expected source[0] title 'First Idea', got %q", params.Sources[0].Title)
	}
	if params.Sources[1].LeafID != "leaf_2" {
		t.Errorf("expected source[1] leaf ID 'leaf_2', got %q", params.Sources[1].LeafID)
	}
	if params.Sources[1].Title != "Second Idea" {
		t.Errorf("expected source[1] title 'Second Idea', got %q", params.Sources[1].Title)
	}
}

func TestStartSynthesis_SeedIDDerivedFromFirstSourceLeaf(t *testing.T) {
	repo := &stubSynthesisRepo{}
	llmProv := &stubLLMProvider{
		chatCompletionFn: func(_ context.Context, _ llm.ChatRequest) (llm.ChatResponse, error) {
			return llm.ChatResponse{Content: validLLMJSON()}, nil
		},
	}

	customSeedID := types.SeedIDFrom("seed_custom")
	leafReader := &stubSourceLeafReader{
		findByIDsFn: func(_ context.Context, _ []types.LeafID) ([]SourceLeaf, error) {
			return []SourceLeaf{
				{ID: types.LeafIDFrom("leaf_1"), SeedID: customSeedID, Title: "A", Summary: "S1"},
				{ID: types.LeafIDFrom("leaf_2"), SeedID: types.SeedIDFrom("seed_other"), Title: "B", Summary: "S2"},
			}, nil
		},
	}
	var seedLookupID types.SeedID
	seedReader := &stubSeedReader{
		findByIDFn: func(_ context.Context, id types.SeedID) (SeedInfo, error) {
			seedLookupID = id
			return SeedInfo{Title: "Custom Seed"}, nil
		},
	}
	leafCreator := &stubLeafCreator{}
	wsMembers := &stubWSMembers{}
	pub := &stubPublisher{}

	svc := newTestService(repo, llmProv, leafReader, seedReader, leafCreator, wsMembers, pub)

	ctx := testCtx("user_test")
	wsID := types.WorkspaceIDFrom("ws_test")
	sourceIDs := []types.LeafID{
		types.LeafIDFrom("leaf_1"),
		types.LeafIDFrom("leaf_2"),
	}

	_, err := svc.StartSynthesis(ctx, wsID, sourceIDs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Seed ID should be derived from the first source leaf.
	if seedLookupID.String() != customSeedID.String() {
		t.Errorf("expected seed lookup for %q, got %q", customSeedID.String(), seedLookupID.String())
	}

	// Leaf creator should receive the same seed ID.
	if leafCreator.lastParams.SeedID.String() != customSeedID.String() {
		t.Errorf("expected leaf params seed ID %q, got %q", customSeedID.String(), leafCreator.lastParams.SeedID.String())
	}
}
