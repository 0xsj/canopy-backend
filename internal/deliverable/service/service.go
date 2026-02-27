package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/0xsj/canopy-backend/internal/deliverable/domain"
	wsdomain "github.com/0xsj/canopy-backend/internal/workspace/domain"
	"github.com/0xsj/canopy-backend/pkg/auth"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/llm"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// WorkspaceMemberReader is a cross-context read port for workspace membership.
type WorkspaceMemberReader interface {
	FindMember(ctx context.Context, workspaceID types.WorkspaceID, userID types.UserID) (wsdomain.WorkspaceMember, error)
}

// SourceLeafReader is a cross-context read port for exploration leaves.
type SourceLeafReader interface {
	FindByIDs(ctx context.Context, ids []types.LeafID) ([]SourceLeaf, error)
}

// SourceLeaf is a cross-context DTO carrying the leaf data needed for deliverable generation.
type SourceLeaf struct {
	ID            types.LeafID
	Title         string
	Summary       string
	KeyPoints     []string
	OpenQuestions []string
	Tags          []string
}

// WorkspaceConfigReader is a cross-context read port for workspace configuration.
type WorkspaceConfigReader interface {
	Configuration(ctx context.Context, workspaceID types.WorkspaceID) (map[string]any, error)
}

const defaultDeliverableTemplate = `Structure the deliverable as follows:
1. Executive Summary — one paragraph overview
2. Key Findings — bullet points of the most important insights
3. Analysis — detailed discussion organized by theme
4. Open Questions — unresolved issues worth further exploration
5. Recommendations — actionable next steps`

// Service implements the deliverable application logic.
type Service struct {
	repo      domain.DeliverableRepository
	wsMembers WorkspaceMemberReader
	llm       llm.ProviderResolver
	leaves    SourceLeafReader
	wsConfig  WorkspaceConfigReader
	pub       events.Publisher
	log       logger.Logger
}

// New creates a new deliverable service.
func New(
	repo domain.DeliverableRepository,
	llm llm.ProviderResolver,
	leaves SourceLeafReader,
	wsConfig WorkspaceConfigReader,
	wsMembers WorkspaceMemberReader,
	pub events.Publisher,
	log logger.Logger,
) *Service {
	return &Service{
		repo:      repo,
		llm:       llm,
		leaves:    leaves,
		wsConfig:  wsConfig,
		wsMembers: wsMembers,
		pub:       pub,
		log:       log,
	}
}

// CreateDraft creates a new deliverable draft from consensus-backed leaves.
func (s *Service) CreateDraft(
	ctx context.Context,
	workspaceID types.WorkspaceID,
	format domain.Format,
	content string,
	sourceLeafIDs []types.LeafID,
) (domain.Deliverable, error) {
	const op = "deliverable: create draft"

	if _, err := s.requireMember(ctx, workspaceID); err != nil {
		return domain.Deliverable{}, canopyerr.Wrap(err, op)
	}

	deliverable, err := domain.NewDeliverable(workspaceID, format, content, sourceLeafIDs)
	if err != nil {
		return domain.Deliverable{}, canopyerr.Wrap(err, op)
	}

	if err := s.repo.Create(ctx, deliverable); err != nil {
		return domain.Deliverable{}, canopyerr.Wrap(err, op)
	}

	leafIDStrs := make([]string, len(sourceLeafIDs))
	for i, id := range sourceLeafIDs {
		leafIDStrs[i] = id.String()
	}

	s.publish(ctx, domain.SubjectDeliverableDraftCreated, workspaceID.String(), domain.DeliverableDraftCreatedData{
		DeliverableID: deliverable.ID().String(),
		WorkspaceID:   workspaceID.String(),
		Format:        string(format),
		SourceLeafIDs: leafIDStrs,
		Timestamp:     time.Now().UTC(),
	})

	s.log.Info("deliverable draft created",
		logger.String("deliverable_id", deliverable.ID().String()),
		logger.String("workspace_id", workspaceID.String()),
	)

	return deliverable, nil
}

// Update edits the deliverable's content. Increments the version.
func (s *Service) Update(ctx context.Context, deliverableID types.DeliverableID, content string) (domain.Deliverable, error) {
	const op = "deliverable: update"

	deliverable, err := s.repo.FindByID(ctx, deliverableID)
	if err != nil {
		return domain.Deliverable{}, canopyerr.Wrap(err, op)
	}

	if _, err := s.requireMember(ctx, deliverable.WorkspaceID()); err != nil {
		return domain.Deliverable{}, canopyerr.Wrap(err, op)
	}

	if err := deliverable.UpdateContent(content); err != nil {
		return domain.Deliverable{}, canopyerr.Wrap(err, op)
	}

	if err := s.repo.Update(ctx, deliverable); err != nil {
		return domain.Deliverable{}, canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectDeliverableUpdated, deliverable.WorkspaceID().String(), domain.DeliverableUpdatedData{
		DeliverableID: deliverableID.String(),
		WorkspaceID:   deliverable.WorkspaceID().String(),
		Version:       deliverable.Version(),
		Timestamp:     time.Now().UTC(),
	})

	return deliverable, nil
}

// Finalize marks the deliverable as finalized. No further edits are expected.
func (s *Service) Finalize(ctx context.Context, deliverableID types.DeliverableID) (domain.Deliverable, error) {
	const op = "deliverable: finalize"

	deliverable, err := s.repo.FindByID(ctx, deliverableID)
	if err != nil {
		return domain.Deliverable{}, canopyerr.Wrap(err, op)
	}

	if _, err := s.requireMember(ctx, deliverable.WorkspaceID()); err != nil {
		return domain.Deliverable{}, canopyerr.Wrap(err, op)
	}

	if err := deliverable.Finalize(); err != nil {
		return domain.Deliverable{}, canopyerr.Wrap(err, op)
	}

	if err := s.repo.Update(ctx, deliverable); err != nil {
		return domain.Deliverable{}, canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectDeliverableFinalized, deliverable.WorkspaceID().String(), domain.DeliverableFinalizedData{
		DeliverableID: deliverableID.String(),
		WorkspaceID:   deliverable.WorkspaceID().String(),
		Format:        string(deliverable.Format()),
		Version:       deliverable.Version(),
		Timestamp:     time.Now().UTC(),
	})

	s.log.Info("deliverable finalized",
		logger.String("deliverable_id", deliverableID.String()),
	)

	return deliverable, nil
}

// FindByID returns a deliverable by ID.
func (s *Service) FindByID(ctx context.Context, id types.DeliverableID) (domain.Deliverable, error) {
	const op = "deliverable: find by id"
	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return domain.Deliverable{}, canopyerr.Wrap(err, op)
	}
	return d, nil
}

// FindByWorkspace returns all deliverables in a workspace.
func (s *Service) FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]domain.Deliverable, error) {
	const op = "deliverable: find by workspace"

	if _, err := s.requireMember(ctx, workspaceID); err != nil {
		return nil, canopyerr.Wrap(err, op)
	}

	deliverables, err := s.repo.FindByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, canopyerr.Wrap(err, op)
	}
	return deliverables, nil
}

// GenerateDeliverable calls the LLM to synthesize source leaves into a coherent
// deliverable document, then persists it as a new draft. The workspace's
// deliverable_template setting (from the configuration JSONB column) is used
// to shape the output format. If no template is configured, a sensible default
// is used.
func (s *Service) GenerateDeliverable(
	ctx context.Context,
	workspaceID types.WorkspaceID,
	format domain.Format,
	sourceLeafIDs []types.LeafID,
) (domain.Deliverable, error) {
	const op = "deliverable: generate"

	if _, err := s.requireMember(ctx, workspaceID); err != nil {
		return domain.Deliverable{}, canopyerr.Wrap(err, op)
	}

	if len(sourceLeafIDs) == 0 {
		return domain.Deliverable{}, canopyerr.Wrap(
			fmt.Errorf("deliverable: at least one source leaf is required"), op)
	}

	// Fetch source leaves.
	leaves, err := s.leaves.FindByIDs(ctx, sourceLeafIDs)
	if err != nil {
		return domain.Deliverable{}, canopyerr.Wrap(err, op)
	}
	if len(leaves) == 0 {
		return domain.Deliverable{}, canopyerr.Wrap(
			fmt.Errorf("deliverable: no leaves found for the given IDs"), op)
	}

	// Read workspace template (best-effort — fall back to default).
	template := defaultDeliverableTemplate
	if s.wsConfig != nil {
		cfg, cfgErr := s.wsConfig.Configuration(ctx, workspaceID)
		if cfgErr == nil {
			if tmpl, ok := cfg["deliverable_template"].(string); ok && tmpl != "" {
				template = tmpl
			}
		}
	}

	// Build prompt and call LLM.
	systemPrompt := buildDeliverablePrompt(leaves, template, string(format))
	provider, model, err := s.llm.Resolve(ctx, workspaceID, llm.TaskSynthesis)
	if err != nil {
		return domain.Deliverable{}, canopyerr.Wrap(fmt.Errorf("resolve llm: %w", err), op)
	}

	resp, err := provider.ChatCompletion(ctx, llm.ChatRequest{
		Model: model,
		Messages: []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: "Generate the deliverable document now."},
		},
		Options: llm.Options{
			MaxTokens:   4096,
			Temperature: 0.4,
		},
	})
	if err != nil {
		return domain.Deliverable{}, canopyerr.Wrap(fmt.Errorf("llm call: %w", err), op)
	}

	content := strings.TrimSpace(resp.Content)
	if content == "" {
		return domain.Deliverable{}, canopyerr.Wrap(fmt.Errorf("llm returned empty content"), op)
	}

	// Persist as a new draft.
	deliverable, err := domain.NewDeliverable(workspaceID, format, content, sourceLeafIDs)
	if err != nil {
		return domain.Deliverable{}, canopyerr.Wrap(err, op)
	}

	if err := s.repo.Create(ctx, deliverable); err != nil {
		return domain.Deliverable{}, canopyerr.Wrap(err, op)
	}

	leafIDStrs := make([]string, len(sourceLeafIDs))
	for i, id := range sourceLeafIDs {
		leafIDStrs[i] = id.String()
	}

	s.publish(ctx, domain.SubjectDeliverableDraftCreated, workspaceID.String(), domain.DeliverableDraftCreatedData{
		DeliverableID: deliverable.ID().String(),
		WorkspaceID:   workspaceID.String(),
		Format:        string(format),
		SourceLeafIDs: leafIDStrs,
		Timestamp:     time.Now().UTC(),
	})

	s.log.Info("deliverable generated via LLM",
		logger.String("deliverable_id", deliverable.ID().String()),
		logger.String("workspace_id", workspaceID.String()),
		logger.Int("source_leaves", len(leaves)),
	)

	return deliverable, nil
}

// buildDeliverablePrompt constructs the system prompt for LLM-based deliverable generation.
func buildDeliverablePrompt(leaves []SourceLeaf, template, format string) string {
	var b strings.Builder

	b.WriteString("You are generating a deliverable document in Canopy, a collaborative thinking platform.\n")
	b.WriteString("Your job is to synthesize the source ideas below into a coherent, well-structured document.\n\n")

	b.WriteString("## Output Format\n")
	b.WriteString(fmt.Sprintf("Produce the document in **%s** format.\n\n", format))

	b.WriteString("## Template Instructions\n")
	b.WriteString(template)
	b.WriteString("\n\n")

	b.WriteString("## Source Ideas\n\n")
	for i, leaf := range leaves {
		b.WriteString(fmt.Sprintf("### Idea %d: %s\n", i+1, leaf.Title))
		if leaf.Summary != "" {
			b.WriteString(fmt.Sprintf("**Summary:** %s\n", leaf.Summary))
		}
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
		if len(leaf.Tags) > 0 {
			b.WriteString(fmt.Sprintf("**Tags:** %s\n", strings.Join(leaf.Tags, ", ")))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Instructions\n")
	b.WriteString("- Synthesize the ideas into a single coherent document following the template above.\n")
	b.WriteString("- Do not simply concatenate the sources — find themes, connections, and insights.\n")
	b.WriteString("- Attribute key ideas to their source when relevant.\n")
	b.WriteString("- Write in a clear, professional tone.\n")
	b.WriteString("- Output ONLY the document content. No meta-commentary.\n")

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
	event.Subject = events.BuildSubject(workspaceID, "deliverable", eventType)
	if pubErr := s.pub.Publish(ctx, event); pubErr != nil {
		s.log.Error("event publish failed", logger.String("type", eventType), logger.Err(pubErr))
	}
}
