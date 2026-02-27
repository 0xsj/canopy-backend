package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/0xsj/canopy-backend/internal/synthesis/domain"
	wsdomain "github.com/0xsj/canopy-backend/internal/workspace/domain"
	"github.com/0xsj/canopy-backend/pkg/auth"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/llm"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// --- Cross-context ports ---

// WorkspaceMemberReader is a cross-context read port for workspace membership.
type WorkspaceMemberReader interface {
	FindMember(ctx context.Context, workspaceID types.WorkspaceID, userID types.UserID) (wsdomain.WorkspaceMember, error)
}

// SourceLeafReader reads leaves from the exploration context for synthesis.
type SourceLeafReader interface {
	FindByIDs(ctx context.Context, ids []types.LeafID) ([]SourceLeaf, error)
}

// SourceLeaf carries the leaf data needed for synthesis prompts.
type SourceLeaf struct {
	ID            types.LeafID
	SeedID        types.SeedID
	Title         string
	Summary       string
	KeyPoints     []string
	OpenQuestions []string
	Tags          []string
}

// SeedReader reads seed data for synthesis prompt context.
type SeedReader interface {
	FindByID(ctx context.Context, id types.SeedID) (SeedInfo, error)
}

// SeedInfo carries the seed data needed for synthesis prompts.
type SeedInfo struct {
	Title       string
	Description string
}

// SynthesisLeafCreator creates the result leaf in the exploration context.
type SynthesisLeafCreator interface {
	Create(ctx context.Context, params SynthesisLeafParams) (types.LeafID, error)
}

// SynthesisLeafParams carries everything needed to create a synthesis leaf.
type SynthesisLeafParams struct {
	WorkspaceID   types.WorkspaceID
	SeedID        types.SeedID
	AuthorID      types.UserID
	Title         string
	Summary       string
	KeyPoints     []string
	OpenQuestions []string
	Tags          []string
	Sources       []LeafSource
}

// LeafSource identifies a source leaf for attribution.
type LeafSource struct {
	LeafID string
	Title  string
}

// Service implements the synthesis application logic.
type Service struct {
	repo        domain.SynthesisRepository
	llm         llm.ProviderResolver
	broadcaster llm.StreamBroadcaster
	leaves      SourceLeafReader
	seeds       SeedReader
	leafCreator SynthesisLeafCreator
	wsMembers   WorkspaceMemberReader
	pub         events.Publisher
	log         logger.Logger
}

// New creates a new synthesis service.
func New(
	repo domain.SynthesisRepository,
	llmResolver llm.ProviderResolver,
	broadcaster llm.StreamBroadcaster,
	leaves SourceLeafReader,
	seeds SeedReader,
	leafCreator SynthesisLeafCreator,
	wsMembers WorkspaceMemberReader,
	pub events.Publisher,
	log logger.Logger,
) *Service {
	return &Service{
		repo:        repo,
		llm:         llmResolver,
		broadcaster: broadcaster,
		leaves:      leaves,
		seeds:       seeds,
		leafCreator: leafCreator,
		wsMembers:   wsMembers,
		pub:         pub,
		log:         log,
	}
}

// StartSynthesis creates a synthesis workflow, publishes the started event, then
// spawns a background goroutine to stream the LLM call and create the result leaf.
// Returns the workflow immediately (status: "in_progress").
func (s *Service) StartSynthesis(ctx context.Context, workspaceID types.WorkspaceID, sourceLeafIDs []types.LeafID) (domain.SynthesisWorkflow, error) {
	const op = "synthesis: start"

	callerID, err := s.requireMember(ctx, workspaceID)
	if err != nil {
		return domain.SynthesisWorkflow{}, canopyerr.Wrap(err, op)
	}

	workflow, err := domain.NewSynthesisWorkflow(workspaceID, callerID, sourceLeafIDs)
	if err != nil {
		return domain.SynthesisWorkflow{}, canopyerr.Wrap(err, op)
	}

	if err := workflow.Start(); err != nil {
		return domain.SynthesisWorkflow{}, canopyerr.Wrap(err, op)
	}

	if err := s.repo.Create(ctx, workflow); err != nil {
		return domain.SynthesisWorkflow{}, canopyerr.Wrap(err, op)
	}

	leafIDStrs := make([]string, len(sourceLeafIDs))
	for i, id := range sourceLeafIDs {
		leafIDStrs[i] = id.String()
	}

	s.publish(ctx, domain.SubjectSynthesisStarted, workspaceID.String(), domain.SynthesisStartedData{
		SynthesisID:   workflow.ID().String(),
		WorkspaceID:   workspaceID.String(),
		InitiatorID:   callerID.String(),
		SourceLeafIDs: leafIDStrs,
		Timestamp:     time.Now().UTC(),
	})

	s.log.Info("synthesis started",
		logger.String("synthesis_id", workflow.ID().String()),
		logger.String("workspace_id", workspaceID.String()),
	)

	// Spawn background goroutine for LLM work.
	go s.runSynthesisStream(context.WithoutCancel(ctx), workflow, workspaceID, callerID, sourceLeafIDs, leafIDStrs)

	return workflow, nil
}

// runSynthesisStream loads sources, streams the LLM call, parses the result,
// creates the synthesis leaf, and completes the workflow. All stream events
// are sent via the broadcaster.
func (s *Service) runSynthesisStream(
	ctx context.Context,
	workflow domain.SynthesisWorkflow,
	workspaceID types.WorkspaceID,
	callerID types.UserID,
	sourceLeafIDs []types.LeafID,
	leafIDStrs []string,
) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	wsID := workspaceID.String()
	streamID := workflow.ID().String()

	_ = s.broadcaster.Send(wsID, llm.StreamMessage{
		Type:     "stream.start",
		StreamID: streamID,
	})

	// Load source leaves.
	sourceLeaves, err := s.leaves.FindByIDs(ctx, sourceLeafIDs)
	if err != nil {
		s.failWorkflowAndNotify(ctx, &workflow, streamID, wsID, fmt.Sprintf("load source leaves: %v", err))
		return
	}
	if len(sourceLeaves) != len(sourceLeafIDs) {
		s.failWorkflowAndNotify(ctx, &workflow, streamID, wsID, fmt.Sprintf("expected %d source leaves, found %d", len(sourceLeafIDs), len(sourceLeaves)))
		return
	}

	seedID := sourceLeaves[0].SeedID

	seedInfo, err := s.seeds.FindByID(ctx, seedID)
	if err != nil {
		s.failWorkflowAndNotify(ctx, &workflow, streamID, wsID, fmt.Sprintf("load seed: %v", err))
		return
	}

	prompt := buildSynthesisLLMPrompt(seedInfo, sourceLeaves)

	provider, model, err := s.llm.Resolve(ctx, workspaceID, llm.TaskSynthesis)
	if err != nil {
		s.failWorkflowAndNotify(ctx, &workflow, streamID, wsID, fmt.Sprintf("resolve llm provider: %v", err))
		return
	}

	req := llm.ChatRequest{
		Model: model,
		Messages: []llm.Message{
			{Role: "system", Content: prompt},
			{Role: "user", Content: "Please synthesize these ideas now."},
		},
	}

	var accumulated strings.Builder

	// Try streaming; fall back to sync.
	sp, ok := provider.(llm.StreamProvider)
	if ok {
		err = sp.ChatCompletionStream(ctx, req, func(chunk llm.StreamChunk) error {
			if chunk.Done {
				return nil
			}
			accumulated.WriteString(chunk.Delta)
			return s.broadcaster.Send(wsID, llm.StreamMessage{
				Type:     "stream.chunk",
				StreamID: streamID,
				Delta:    chunk.Delta,
			})
		})
	} else {
		var resp llm.ChatResponse
		resp, err = provider.ChatCompletion(ctx, req)
		if err == nil {
			accumulated.WriteString(resp.Content)
		}
	}

	if err != nil {
		s.failWorkflowAndNotify(ctx, &workflow, streamID, wsID, fmt.Sprintf("llm call: %v", err))
		return
	}

	content := accumulated.String()

	// Parse structured response.
	result, err := parseSynthesisResponse(content)
	if err != nil {
		s.failWorkflowAndNotify(ctx, &workflow, streamID, wsID, fmt.Sprintf("parse llm response: %v", err))
		return
	}

	// Build source attribution.
	sources := make([]LeafSource, len(sourceLeaves))
	for i, leaf := range sourceLeaves {
		sources[i] = LeafSource{LeafID: leaf.ID.String(), Title: leaf.Title}
	}

	// Create synthesis leaf.
	leafID, err := s.leafCreator.Create(ctx, SynthesisLeafParams{
		WorkspaceID:   workspaceID,
		SeedID:        seedID,
		AuthorID:      callerID,
		Title:         result.Title,
		Summary:       result.Summary,
		KeyPoints:     result.KeyPoints,
		OpenQuestions: result.OpenQuestions,
		Tags:          result.Tags,
		Sources:       sources,
	})
	if err != nil {
		s.failWorkflowAndNotify(ctx, &workflow, streamID, wsID, fmt.Sprintf("create synthesis leaf: %v", err))
		return
	}

	// Complete workflow.
	if err := workflow.Complete(leafID); err != nil {
		s.log.Error("synthesis: complete workflow failed", logger.Err(err))
		return
	}
	if err := s.repo.Update(ctx, workflow); err != nil {
		s.log.Error("synthesis: persist completed workflow failed", logger.Err(err))
		return
	}

	s.publish(ctx, domain.SubjectSynthesisCompleted, wsID, domain.SynthesisCompletedData{
		SynthesisID:  streamID,
		WorkspaceID:  wsID,
		ResultLeafID: leafID.String(),
		Timestamp:    time.Now().UTC(),
	})

	s.publish(ctx, domain.SubjectSynthesisLeafCreated, wsID, domain.SynthesisLeafCreatedData{
		LeafID:        leafID.String(),
		WorkspaceID:   wsID,
		SourceLeafIDs: leafIDStrs,
		InitiatorID:   callerID.String(),
		Timestamp:     time.Now().UTC(),
	})

	_ = s.broadcaster.Send(wsID, llm.StreamMessage{
		Type:     "stream.end",
		StreamID: streamID,
		Content:  content,
	})

	s.log.Info("synthesis completed",
		logger.String("synthesis_id", streamID),
		logger.String("result_leaf_id", leafID.String()),
	)
}

// failWorkflowAndNotify fails the workflow, persists it, and sends a stream.error.
func (s *Service) failWorkflowAndNotify(ctx context.Context, workflow *domain.SynthesisWorkflow, streamID, wsID, reason string) {
	s.failWorkflow(ctx, workflow, reason)
	_ = s.broadcaster.Send(wsID, llm.StreamMessage{
		Type:     "stream.error",
		StreamID: streamID,
		Error:    reason,
	})
}

// CompleteSynthesis marks a workflow as completed with the resulting leaf.
// This is a manual override — normally StartSynthesis completes automatically.
func (s *Service) CompleteSynthesis(ctx context.Context, synthesisID domain.SynthesisID, resultLeafID types.LeafID) (domain.SynthesisWorkflow, error) {
	const op = "synthesis: complete"

	workflow, err := s.repo.FindByID(ctx, synthesisID)
	if err != nil {
		return domain.SynthesisWorkflow{}, canopyerr.Wrap(err, op)
	}

	if err := workflow.Complete(resultLeafID); err != nil {
		return domain.SynthesisWorkflow{}, canopyerr.Wrap(err, op)
	}

	if err := s.repo.Update(ctx, workflow); err != nil {
		return domain.SynthesisWorkflow{}, canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectSynthesisCompleted, workflow.WorkspaceID().String(), domain.SynthesisCompletedData{
		SynthesisID:  synthesisID.String(),
		WorkspaceID:  workflow.WorkspaceID().String(),
		ResultLeafID: resultLeafID.String(),
		Timestamp:    time.Now().UTC(),
	})

	s.log.Info("synthesis completed",
		logger.String("synthesis_id", synthesisID.String()),
		logger.String("result_leaf_id", resultLeafID.String()),
	)

	return workflow, nil
}

// FailSynthesis marks a workflow as failed.
func (s *Service) FailSynthesis(ctx context.Context, synthesisID domain.SynthesisID, reason string) error {
	const op = "synthesis: fail"

	workflow, err := s.repo.FindByID(ctx, synthesisID)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := workflow.Fail(reason); err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := s.repo.Update(ctx, workflow); err != nil {
		return canopyerr.Wrap(err, op)
	}

	s.log.Warn("synthesis failed",
		logger.String("synthesis_id", synthesisID.String()),
		logger.String("reason", reason),
	)

	return nil
}

// FindByWorkspace returns all synthesis workflows in a workspace.
func (s *Service) FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]domain.SynthesisWorkflow, error) {
	const op = "synthesis: find by workspace"

	if _, err := s.requireMember(ctx, workspaceID); err != nil {
		return nil, canopyerr.Wrap(err, op)
	}

	workflows, err := s.repo.FindByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, canopyerr.Wrap(err, op)
	}
	return workflows, nil
}

// --- LLM Helpers ---

// failWorkflow marks a workflow as failed and persists it.
func (s *Service) failWorkflow(ctx context.Context, workflow *domain.SynthesisWorkflow, reason string) {
	if err := workflow.Fail(reason); err != nil {
		s.log.Error("synthesis: failed to transition workflow to failed state",
			logger.String("synthesis_id", workflow.ID().String()),
			logger.Err(err),
		)
		return
	}
	if err := s.repo.Update(ctx, *workflow); err != nil {
		s.log.Error("synthesis: failed to persist failed workflow",
			logger.String("synthesis_id", workflow.ID().String()),
			logger.Err(err),
		)
	}
	s.log.Warn("synthesis failed",
		logger.String("synthesis_id", workflow.ID().String()),
		logger.String("reason", reason),
	)
}

// synthesisResult is the structured JSON output expected from the LLM.
type synthesisResult struct {
	Title         string   `json:"title"`
	Summary       string   `json:"summary"`
	KeyPoints     []string `json:"key_points"`
	OpenQuestions []string `json:"open_questions"`
	Tags          []string `json:"tags"`
}

// parseSynthesisResponse extracts structured leaf data from the LLM response.
func parseSynthesisResponse(content string) (synthesisResult, error) {
	content = strings.TrimSpace(content)

	// LLM may wrap JSON in markdown fences — extract the JSON object.
	if idx := strings.Index(content, "{"); idx >= 0 {
		if end := strings.LastIndex(content, "}"); end > idx {
			content = content[idx : end+1]
		}
	}

	var result synthesisResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return synthesisResult{}, fmt.Errorf("invalid JSON: %w", err)
	}

	if result.Title == "" {
		return synthesisResult{}, fmt.Errorf("synthesis result missing title")
	}
	if result.Summary == "" {
		return synthesisResult{}, fmt.Errorf("synthesis result missing summary")
	}

	return result, nil
}

// buildSynthesisLLMPrompt constructs the system prompt for the synthesis LLM call.
func buildSynthesisLLMPrompt(seed SeedInfo, leaves []SourceLeaf) string {
	var b strings.Builder

	b.WriteString("You are synthesizing multiple ideas into a unified insight in Canopy, a collaborative thinking platform.\n\n")
	b.WriteString("## Seed Topic\n")
	b.WriteString(fmt.Sprintf("**Title:** %s\n", seed.Title))
	if seed.Description != "" {
		b.WriteString(fmt.Sprintf("**Description:** %s\n", seed.Description))
	}

	b.WriteString("\n## Source Ideas\n\n")
	for i, leaf := range leaves {
		b.WriteString(fmt.Sprintf("### Idea %d: %s\n", i+1, leaf.Title))
		b.WriteString(fmt.Sprintf("**Summary:** %s\n", leaf.Summary))
		if len(leaf.KeyPoints) > 0 {
			b.WriteString("**Key Points:**\n")
			for _, kp := range leaf.KeyPoints {
				b.WriteString(fmt.Sprintf("- %s\n", kp))
			}
		}
		if len(leaf.OpenQuestions) > 0 {
			b.WriteString("**Open Questions:**\n")
			for _, q := range leaf.OpenQuestions {
				b.WriteString(fmt.Sprintf("- %s\n", q))
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("## Your Task\n")
	b.WriteString("Synthesize these ideas into a single, coherent insight. Find the deeper connections ")
	b.WriteString("between them. Resolve any tensions or contradictions. Preserve the strengths of each source.\n\n")
	b.WriteString("Respond with ONLY a JSON object in this exact format (no markdown fences, no extra text):\n")
	b.WriteString("{\n")
	b.WriteString("  \"title\": \"A clear, concise title for the synthesized insight\",\n")
	b.WriteString("  \"summary\": \"A paragraph capturing the unified understanding\",\n")
	b.WriteString("  \"key_points\": [\"3-5 key points from the synthesis\"],\n")
	b.WriteString("  \"open_questions\": [\"Questions that emerge from combining these ideas\"],\n")
	b.WriteString("  \"tags\": [\"relevant category tags\"]\n")
	b.WriteString("}\n")

	return b.String()
}

// --- Auth Helpers ---

func (s *Service) requireMember(ctx context.Context, workspaceID types.WorkspaceID) (types.UserID, error) {
	claims, ok := auth.FromClaims(ctx)
	if !ok {
		return types.UserID{}, canopyerr.ErrUnauthenticated
	}
	callerID := types.UserIDFrom(claims.Subject)

	if _, err := s.wsMembers.FindMember(ctx, workspaceID, callerID); err != nil {
		if canopyerr.GetKind(err) == canopyerr.KindNotFound {
			return types.UserID{}, canopyerr.ErrUnauthorized
		}
		return types.UserID{}, err
	}
	return callerID, nil
}

// --- Event Publishing ---

func (s *Service) publish(ctx context.Context, eventType, workspaceID string, data any) {
	event, err := events.New(eventType, workspaceID, data)
	if err != nil {
		s.log.Error("event creation failed", logger.String("type", eventType), logger.Err(err))
		return
	}
	event.Subject = events.BuildSubject(workspaceID, "synthesis", eventType)
	if pubErr := s.pub.Publish(ctx, event); pubErr != nil {
		s.log.Error("event publish failed", logger.String("type", eventType), logger.Err(pubErr))
	}
}
