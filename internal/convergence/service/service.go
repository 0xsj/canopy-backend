package service

import (
	"context"
	"time"

	"github.com/0xsj/canopy-backend/internal/convergence/domain"
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

// LeafWriter is a cross-context write port for promoting leaves.
type LeafWriter interface {
	UpdateLayer(ctx context.Context, id types.LeafID, layer string) error
}

// Service implements the convergence application logic.
type Service struct {
	signals     domain.SignalRepository
	checkpoints domain.CheckpointRepository
	leafWriter  LeafWriter
	wsMembers   WorkspaceMemberReader
	pub         events.Publisher
	log         logger.Logger
}

// New creates a new convergence service.
func New(
	signals domain.SignalRepository,
	checkpoints domain.CheckpointRepository,
	leafWriter LeafWriter,
	wsMembers WorkspaceMemberReader,
	pub events.Publisher,
	log logger.Logger,
) *Service {
	return &Service{
		signals:     signals,
		checkpoints: checkpoints,
		leafWriter:  leafWriter,
		wsMembers:   wsMembers,
		pub:         pub,
		log:         log,
	}
}

// RecordSignal records a user's signal on a leaf. Caller must be a workspace member.
func (s *Service) RecordSignal(
	ctx context.Context,
	workspaceID types.WorkspaceID,
	leafID types.LeafID,
	signalType domain.SignalType,
	annotation string,
) (domain.Signal, error) {
	const op = "convergence: record signal"

	callerID, err := s.requireMember(ctx, workspaceID)
	if err != nil {
		return domain.Signal{}, canopyerr.Wrap(err, op)
	}

	signal, err := domain.NewSignal(workspaceID, leafID, callerID, signalType, annotation)
	if err != nil {
		return domain.Signal{}, canopyerr.Wrap(err, op)
	}

	if err := s.signals.Create(ctx, signal); err != nil {
		return domain.Signal{}, canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectConsensusSignalRecorded, workspaceID.String(), domain.ConsensusSignalRecordedData{
		CheckpointID: "",
		WorkspaceID:  workspaceID.String(),
		UserID:       callerID.String(),
		Position:     string(signalType),
		Timestamp:    time.Now().UTC(),
	})

	s.log.Info("signal recorded",
		logger.String("leaf_id", leafID.String()),
		logger.String("type", string(signalType)),
	)

	return signal, nil
}

// RemoveSignal retracts a user's signal. Caller must be the signal owner.
func (s *Service) RemoveSignal(ctx context.Context, signalID domain.SignalID) error {
	const op = "convergence: remove signal"

	_, ok := auth.FromClaims(ctx)
	if !ok {
		return canopyerr.Wrap(canopyerr.ErrUnauthenticated, op)
	}

	if err := s.signals.Delete(ctx, signalID); err != nil {
		return canopyerr.Wrap(err, op)
	}

	return nil
}

// GetSignalCounts returns signal counts grouped by type for a leaf.
func (s *Service) GetSignalCounts(ctx context.Context, leafID types.LeafID) (map[domain.SignalType]int, error) {
	const op = "convergence: get signal counts"
	counts, err := s.signals.CountByLeaf(ctx, leafID)
	if err != nil {
		return nil, canopyerr.Wrap(err, op)
	}
	return counts, nil
}

// FindUserSignals returns all signals by a user in a workspace.
func (s *Service) FindUserSignals(ctx context.Context, workspaceID types.WorkspaceID, userID types.UserID) ([]domain.Signal, error) {
	const op = "convergence: find user signals"
	signals, err := s.signals.FindByUser(ctx, workspaceID, userID)
	if err != nil {
		return nil, canopyerr.Wrap(err, op)
	}
	return signals, nil
}

// CreateCheckpoint creates a new consensus checkpoint for a set of leaves.
func (s *Service) CreateCheckpoint(ctx context.Context, workspaceID types.WorkspaceID, leafIDs []types.LeafID) (domain.Checkpoint, error) {
	const op = "convergence: create checkpoint"

	if _, err := s.requireMember(ctx, workspaceID); err != nil {
		return domain.Checkpoint{}, canopyerr.Wrap(err, op)
	}

	checkpoint, err := domain.NewCheckpoint(workspaceID, leafIDs)
	if err != nil {
		return domain.Checkpoint{}, canopyerr.Wrap(err, op)
	}

	if err := s.checkpoints.Create(ctx, checkpoint); err != nil {
		return domain.Checkpoint{}, canopyerr.Wrap(err, op)
	}

	leafIDStrs := make([]string, len(leafIDs))
	for i, id := range leafIDs {
		leafIDStrs[i] = id.String()
	}

	s.publish(ctx, domain.SubjectCheckpointCreated, workspaceID.String(), domain.CheckpointCreatedData{
		CheckpointID: checkpoint.ID().String(),
		WorkspaceID:  workspaceID.String(),
		LeafIDs:      leafIDStrs,
		Timestamp:    time.Now().UTC(),
	})

	s.log.Info("checkpoint created",
		logger.String("checkpoint_id", checkpoint.ID().String()),
		logger.String("workspace_id", workspaceID.String()),
	)

	return checkpoint, nil
}

// ResolveCheckpoint resolves a checkpoint. If consensus is reached, promotes
// the checkpoint's leaves to the canopy layer.
func (s *Service) ResolveCheckpoint(ctx context.Context, checkpointID types.CheckpointID) error {
	const op = "convergence: resolve checkpoint"

	checkpoint, err := s.checkpoints.FindByID(ctx, checkpointID)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := checkpoint.Resolve(); err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := s.checkpoints.Update(ctx, checkpoint); err != nil {
		return canopyerr.Wrap(err, op)
	}

	if checkpoint.HasConsensus() {
		// Promote all leaves to canopy.
		for _, leafID := range checkpoint.LeafIDs() {
			if err := s.leafWriter.UpdateLayer(ctx, leafID, "canopy"); err != nil {
				s.log.Error("leaf promotion failed",
					logger.String("leaf_id", leafID.String()),
					logger.Err(err),
				)
				continue
			}

			s.publish(ctx, domain.SubjectLeafPromotedToCanopy, checkpoint.WorkspaceID().String(), domain.LeafPromotedToCanopyData{
				LeafID:      leafID.String(),
				WorkspaceID: checkpoint.WorkspaceID().String(),
				Timestamp:   time.Now().UTC(),
			})
		}

		leafIDStrs := make([]string, len(checkpoint.LeafIDs()))
		for i, id := range checkpoint.LeafIDs() {
			leafIDStrs[i] = id.String()
		}

		s.publish(ctx, domain.SubjectConsensusReached, checkpoint.WorkspaceID().String(), domain.ConsensusReachedData{
			CheckpointID: checkpointID.String(),
			WorkspaceID:  checkpoint.WorkspaceID().String(),
			LeafIDs:      leafIDStrs,
			Timestamp:    time.Now().UTC(),
		})

		s.log.Info("consensus reached",
			logger.String("checkpoint_id", checkpointID.String()),
		)
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
	event.Subject = events.BuildSubject(workspaceID, "convergence", eventType)
	if pubErr := s.pub.Publish(ctx, event); pubErr != nil {
		s.log.Error("event publish failed", logger.String("type", eventType), logger.Err(pubErr))
	}
}
