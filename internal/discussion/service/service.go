package service

import (
	"context"
	"time"

	"github.com/0xsj/canopy-backend/internal/discussion/domain"
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

// Service implements the discussion application logic.
type Service struct {
	threads   domain.ThreadRepository
	wsMembers WorkspaceMemberReader
	pub       events.Publisher
	log       logger.Logger
}

// New creates a new discussion service.
func New(threads domain.ThreadRepository, wsMembers WorkspaceMemberReader, pub events.Publisher, log logger.Logger) *Service {
	return &Service{threads: threads, wsMembers: wsMembers, pub: pub, log: log}
}

// AddComment appends a comment to a leaf's thread. Creates the thread if it
// doesn't exist. workspaceID is needed for membership validation.
func (s *Service) AddComment(ctx context.Context, workspaceID types.WorkspaceID, leafID types.LeafID, content string) error {
	const op = "discussion: add comment"

	callerID, err := s.requireMember(ctx, workspaceID)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	comment, err := domain.NewComment(callerID, content)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	thread, err := s.threads.FindByLeaf(ctx, leafID)
	if err != nil {
		if canopyerr.GetKind(err) != canopyerr.KindNotFound {
			return canopyerr.Wrap(err, op)
		}
		thread, err = domain.NewThread(leafID)
		if err != nil {
			return canopyerr.Wrap(err, op)
		}
	}

	thread.AddComment(comment)

	if err := s.threads.Save(ctx, thread); err != nil {
		return canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectThreadCommentAdded, workspaceID.String(), domain.ThreadCommentAddedData{
		LeafID:      leafID.String(),
		AuthorID:    callerID.String(),
		CommentText: content,
		Timestamp:   time.Now().UTC(),
	})

	return nil
}

// FindThread returns the discussion thread for a leaf.
func (s *Service) FindThread(ctx context.Context, leafID types.LeafID) (domain.Thread, error) {
	const op = "discussion: find thread"
	thread, err := s.threads.FindByLeaf(ctx, leafID)
	if err != nil {
		return domain.Thread{}, canopyerr.Wrap(err, op)
	}
	return thread, nil
}

// EnsureThread creates an empty thread for a leaf if one doesn't exist.
// Called by event subscribers when a new leaf is created.
func (s *Service) EnsureThread(ctx context.Context, leafID types.LeafID) error {
	const op = "discussion: ensure thread"

	exists, err := s.threads.Exists(ctx, leafID)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}
	if exists {
		return nil
	}

	thread, err := domain.NewThread(leafID)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := s.threads.Save(ctx, thread); err != nil {
		return canopyerr.Wrap(err, op)
	}

	s.log.Info("thread created for leaf", logger.String("leaf_id", leafID.String()))
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
	event.Subject = events.BuildSubject(workspaceID, "discussion", eventType)
	if pubErr := s.pub.Publish(ctx, event); pubErr != nil {
		s.log.Error("event publish failed", logger.String("type", eventType), logger.Err(pubErr))
	}
}
