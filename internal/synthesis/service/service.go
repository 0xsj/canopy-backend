package service

import (
	"context"
	"time"

	"github.com/0xsj/canopy-backend/internal/synthesis/domain"
	wsdomain "github.com/0xsj/canopy-backend/internal/workspace/domain"
	"github.com/0xsj/canopy-backend/pkg/auth"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// WorkspaceMemberReader is a cross-context read port for workspace membership.
type WorkspaceMemberReader interface {
	FindMember(ctx context.Context, workspaceID types.WorkspaceID, userID types.UserID) (wsdomain.WorkspaceMember, error)
}

// Service implements the synthesis application logic.
type Service struct {
	repo      domain.SynthesisRepository
	wsMembers WorkspaceMemberReader
	pub       events.Publisher
	log       logger.Logger
}

// New creates a new synthesis service.
func New(repo domain.SynthesisRepository, wsMembers WorkspaceMemberReader, pub events.Publisher, log logger.Logger) *Service {
	return &Service{repo: repo, wsMembers: wsMembers, pub: pub, log: log}
}

// StartSynthesis creates a new synthesis workflow. Caller must be a workspace member.
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

	return workflow, nil
}

// CompleteSynthesis marks a workflow as completed with the resulting leaf.
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
