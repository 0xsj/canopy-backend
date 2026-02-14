package service

import (
	"context"
	"time"

	"github.com/0xsj/canopy-backend/internal/deliverable/domain"
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

// Service implements the deliverable application logic.
type Service struct {
	repo      domain.DeliverableRepository
	wsMembers WorkspaceMemberReader
	pub       events.Publisher
	log       logger.Logger
}

// New creates a new deliverable service.
func New(repo domain.DeliverableRepository, wsMembers WorkspaceMemberReader, pub events.Publisher, log logger.Logger) *Service {
	return &Service{repo: repo, wsMembers: wsMembers, pub: pub, log: log}
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
