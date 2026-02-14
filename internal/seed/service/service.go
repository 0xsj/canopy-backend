package service

import (
	"context"
	"time"

	"github.com/0xsj/canopy-backend/internal/seed/domain"
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

// Service implements the seed application logic.
type Service struct {
	repo      domain.SeedRepository
	wsMembers WorkspaceMemberReader
	pub       events.Publisher
	log       logger.Logger
}

// New creates a new seed service.
func New(repo domain.SeedRepository, wsMembers WorkspaceMemberReader, pub events.Publisher, log logger.Logger) *Service {
	return &Service{repo: repo, wsMembers: wsMembers, pub: pub, log: log}
}

// Plant creates a new seed in a workspace. Caller must be a workspace member.
func (s *Service) Plant(ctx context.Context, workspaceID types.WorkspaceID, title, description string) (domain.Seed, error) {
	const op = "seed: plant"

	callerID, err := s.requireMember(ctx, workspaceID)
	if err != nil {
		return domain.Seed{}, canopyerr.Wrap(err, op)
	}

	seed, err := domain.NewSeed(workspaceID, callerID, title, description)
	if err != nil {
		return domain.Seed{}, canopyerr.Wrap(err, op)
	}

	if err := s.repo.Create(ctx, seed); err != nil {
		return domain.Seed{}, canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectSeedPlanted, workspaceID.String(), domain.SeedPlantedData{
		SeedID:      seed.ID().String(),
		WorkspaceID: workspaceID.String(),
		AuthorID:    callerID.String(),
		Title:       seed.Title(),
		Timestamp:   time.Now().UTC(),
	})

	s.log.Info("seed planted",
		logger.String("seed_id", seed.ID().String()),
		logger.String("workspace_id", workspaceID.String()),
	)

	return seed, nil
}

// FindByID returns a seed by ID.
func (s *Service) FindByID(ctx context.Context, id types.SeedID) (domain.Seed, error) {
	const op = "seed: find by id"
	seed, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return domain.Seed{}, canopyerr.Wrap(err, op)
	}
	return seed, nil
}

// FindByWorkspace returns all seeds in a workspace. Caller must be a member.
func (s *Service) FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]domain.Seed, error) {
	const op = "seed: find by workspace"

	if _, err := s.requireMember(ctx, workspaceID); err != nil {
		return nil, canopyerr.Wrap(err, op)
	}

	seeds, err := s.repo.FindByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, canopyerr.Wrap(err, op)
	}
	return seeds, nil
}

// UpdateConstraints updates a seed's constraints. Caller must be a workspace member.
func (s *Service) UpdateConstraints(ctx context.Context, seedID types.SeedID, constraints map[string]any) (domain.Seed, error) {
	const op = "seed: update constraints"

	seed, err := s.repo.FindByID(ctx, seedID)
	if err != nil {
		return domain.Seed{}, canopyerr.Wrap(err, op)
	}

	if _, err := s.requireMember(ctx, seed.WorkspaceID()); err != nil {
		return domain.Seed{}, canopyerr.Wrap(err, op)
	}

	seed.UpdateConstraints(constraints)

	if err := s.repo.Update(ctx, seed); err != nil {
		return domain.Seed{}, canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectSeedConstraintsUpdated, seed.WorkspaceID().String(), domain.SeedConstraintsUpdatedData{
		SeedID:      seedID.String(),
		WorkspaceID: seed.WorkspaceID().String(),
		Timestamp:   time.Now().UTC(),
	})

	return seed, nil
}

// UpdatePosition updates a seed's canvas position. Caller must be a workspace member.
func (s *Service) UpdatePosition(ctx context.Context, seedID types.SeedID, x, y float64) error {
	const op = "seed: update position"

	seed, err := s.repo.FindByID(ctx, seedID)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	if _, err := s.requireMember(ctx, seed.WorkspaceID()); err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := s.repo.UpdatePosition(ctx, seedID, x, y); err != nil {
		return canopyerr.Wrap(err, op)
	}

	return nil
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
	event.Subject = events.BuildSubject(workspaceID, "seed", eventType)
	if pubErr := s.pub.Publish(ctx, event); pubErr != nil {
		s.log.Error("event publish failed", logger.String("type", eventType), logger.Err(pubErr))
	}
}
