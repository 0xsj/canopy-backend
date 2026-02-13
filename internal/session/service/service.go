package service

import (
	"context"
	"time"

	"github.com/0xsj/canopy-backend/internal/session/domain"
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

// Service implements the session application logic.
type Service struct {
	sessions  domain.SessionRepository
	assembler domain.ContextAssembler
	wsMembers WorkspaceMemberReader
	pub       events.Publisher
	log       logger.Logger
}

// New creates a new session service.
func New(
	sessions domain.SessionRepository,
	assembler domain.ContextAssembler,
	wsMembers WorkspaceMemberReader,
	pub events.Publisher,
	log logger.Logger,
) *Service {
	return &Service{
		sessions:  sessions,
		assembler: assembler,
		wsMembers: wsMembers,
		pub:       pub,
		log:       log,
	}
}

// StartSession creates a new AI-assisted session. Caller must be a workspace member.
func (s *Service) StartSession(
	ctx context.Context,
	workspaceID types.WorkspaceID,
	seedID types.SeedID,
	parentLeafID types.LeafID,
	sessionType domain.SessionType,
) (domain.Session, error) {
	const op = "session: start"

	callerID, err := s.requireMember(ctx, workspaceID)
	if err != nil {
		return domain.Session{}, canopyerr.Wrap(err, op)
	}

	session, err := domain.NewSession(workspaceID, callerID, seedID, parentLeafID, sessionType)
	if err != nil {
		return domain.Session{}, canopyerr.Wrap(err, op)
	}

	if err := s.sessions.Create(ctx, session); err != nil {
		return domain.Session{}, canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectSessionStarted, workspaceID.String(), domain.SessionStartedData{
		SessionID:   session.ID().String(),
		WorkspaceID: workspaceID.String(),
		UserID:      callerID.String(),
		SeedID:      seedID.String(),
		SessionType: string(sessionType),
		Timestamp:   time.Now().UTC(),
	})

	s.log.Info("session started",
		logger.String("session_id", session.ID().String()),
		logger.String("type", string(sessionType)),
	)

	return session, nil
}

// AddMessage appends a message to a session.
func (s *Service) AddMessage(ctx context.Context, sessionID domain.SessionID, msg domain.Message) (domain.Session, error) {
	const op = "session: add message"

	session, err := s.sessions.FindByID(ctx, sessionID)
	if err != nil {
		return domain.Session{}, canopyerr.Wrap(err, op)
	}

	if err := session.AddMessage(msg); err != nil {
		return domain.Session{}, canopyerr.Wrap(err, op)
	}

	if err := s.sessions.Update(ctx, session); err != nil {
		return domain.Session{}, canopyerr.Wrap(err, op)
	}

	return session, nil
}

// Checkpoint transitions a session from active to the shaping phase.
func (s *Service) Checkpoint(ctx context.Context, sessionID domain.SessionID) (domain.Session, error) {
	const op = "session: checkpoint"

	session, err := s.sessions.FindByID(ctx, sessionID)
	if err != nil {
		return domain.Session{}, canopyerr.Wrap(err, op)
	}

	if err := session.EnterCheckpoint(); err != nil {
		return domain.Session{}, canopyerr.Wrap(err, op)
	}

	if err := s.sessions.Update(ctx, session); err != nil {
		return domain.Session{}, canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectSessionCheckpointed, session.WorkspaceID().String(), domain.SessionCheckpointedData{
		SessionID:   sessionID.String(),
		WorkspaceID: session.WorkspaceID().String(),
		UserID:      session.UserID().String(),
		Timestamp:   time.Now().UTC(),
	})

	return session, nil
}

// CompleteSession marks a session as successfully completed.
func (s *Service) CompleteSession(ctx context.Context, sessionID domain.SessionID, leafID types.LeafID) (domain.Session, error) {
	const op = "session: complete"

	session, err := s.sessions.FindByID(ctx, sessionID)
	if err != nil {
		return domain.Session{}, canopyerr.Wrap(err, op)
	}

	if err := session.Complete(); err != nil {
		return domain.Session{}, canopyerr.Wrap(err, op)
	}

	if err := s.sessions.Update(ctx, session); err != nil {
		return domain.Session{}, canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectSessionCompleted, session.WorkspaceID().String(), domain.SessionCompletedData{
		SessionID:   sessionID.String(),
		WorkspaceID: session.WorkspaceID().String(),
		UserID:      session.UserID().String(),
		LeafID:      leafID.String(),
		Timestamp:   time.Now().UTC(),
	})

	s.log.Info("session completed",
		logger.String("session_id", sessionID.String()),
		logger.String("leaf_id", leafID.String()),
	)

	return session, nil
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
	event.Subject = events.BuildSubject(workspaceID, "session", eventType)
	if pubErr := s.pub.Publish(ctx, event); pubErr != nil {
		s.log.Error("event publish failed", logger.String("type", eventType), logger.Err(pubErr))
	}
}
