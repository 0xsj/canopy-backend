package service

import (
	"context"
	"time"

	"github.com/0xsj/canopy-backend/internal/session/domain"
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

// Service implements the session application logic.
type Service struct {
	sessions  domain.SessionRepository
	assembler domain.ContextAssembler
	llm       llm.ProviderResolver
	wsMembers WorkspaceMemberReader
	pub       events.Publisher
	log       logger.Logger
}

// New creates a new session service.
func New(
	sessions domain.SessionRepository,
	assembler domain.ContextAssembler,
	llmResolver llm.ProviderResolver,
	wsMembers WorkspaceMemberReader,
	pub events.Publisher,
	log logger.Logger,
) *Service {
	return &Service{
		sessions:  sessions,
		assembler: assembler,
		llm:       llmResolver,
		wsMembers: wsMembers,
		pub:       pub,
		log:       log,
	}
}

// GetSession returns a single session by ID. Caller must be a member of the session's workspace.
func (s *Service) GetSession(ctx context.Context, sessionID domain.SessionID) (domain.Session, error) {
	const op = "session: get"

	session, err := s.sessions.FindByID(ctx, sessionID)
	if err != nil {
		return domain.Session{}, canopyerr.Wrap(err, op)
	}

	if _, err := s.requireMember(ctx, session.WorkspaceID()); err != nil {
		return domain.Session{}, canopyerr.Wrap(err, op)
	}

	return session, nil
}

// ListSessions returns all sessions for a workspace. Caller must be a workspace member.
func (s *Service) ListSessions(ctx context.Context, workspaceID types.WorkspaceID) ([]domain.Session, error) {
	const op = "session: list"

	if _, err := s.requireMember(ctx, workspaceID); err != nil {
		return nil, canopyerr.Wrap(err, op)
	}

	sessions, err := s.sessions.FindByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, canopyerr.Wrap(err, op)
	}

	return sessions, nil
}

// StartSession creates a new AI-assisted session. Caller must be a workspace member.
func (s *Service) StartSession(
	ctx context.Context,
	workspaceID types.WorkspaceID,
	seedID types.SeedID,
	parentLeafID types.LeafID,
	sourceLeafIDs []types.LeafID,
	sessionType domain.SessionType,
) (domain.Session, error) {
	const op = "session: start"

	callerID, err := s.requireMember(ctx, workspaceID)
	if err != nil {
		return domain.Session{}, canopyerr.Wrap(err, op)
	}

	session, err := domain.NewSession(workspaceID, callerID, seedID, parentLeafID, sourceLeafIDs, sessionType)
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

// AddMessage appends a user message to a session, calls the LLM, and appends the assistant response.
// Idempotent: if the last user message has the same content, returns the session as-is.
func (s *Service) AddMessage(ctx context.Context, sessionID domain.SessionID, msg domain.Message) (domain.Session, error) {
	const op = "session: add message"

	session, err := s.sessions.FindByID(ctx, sessionID)
	if err != nil {
		return domain.Session{}, canopyerr.Wrap(err, op)
	}

	// Idempotency guard: if the most recent message with the same role has
	// identical content, this is a duplicate (e.g. client retried after timeout).
	// Check the last TWO messages to catch both cases:
	//   - Last msg is user (LLM still running): last.Role == msg.Role
	//   - Last msg is assistant (LLM completed): second-to-last is the user msg
	if msgs := session.Messages(); len(msgs) > 0 {
		for i := len(msgs) - 1; i >= 0 && i >= len(msgs)-2; i-- {
			if msgs[i].Role == msg.Role && msgs[i].Content == msg.Content {
				s.log.Info("duplicate message skipped",
					logger.String("session_id", sessionID.String()),
					logger.String("role", msg.Role),
				)
				return session, nil
			}
		}
	}

	// Append the user message.
	if err := session.AddMessage(msg); err != nil {
		return domain.Session{}, canopyerr.Wrap(err, op)
	}

	if err := s.sessions.Update(ctx, session); err != nil {
		return domain.Session{}, canopyerr.Wrap(err, op)
	}

	// Assemble context and call LLM.
	// Use a detached context for the LLM call so it completes even if the
	// HTTP client disconnects (prevents orphaned user messages without responses).
	llmCtx, llmCancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Minute)
	defer llmCancel()

	assembled, err := s.assembler.Assemble(llmCtx, session)
	if err != nil {
		s.log.Error("context assembly failed", logger.Err(err))
		return session, canopyerr.Wrap(err, op)
	}

	llmMessages := make([]llm.Message, len(assembled))
	for i, m := range assembled {
		llmMessages[i] = llm.Message{Role: m.Role, Content: m.Content}
	}

	provider, err := s.llm.Resolve(llmCtx, session.WorkspaceID())
	if err != nil {
		s.log.Error("llm resolve failed", logger.Err(err))
		return session, canopyerr.Wrap(err, op)
	}

	resp, err := provider.ChatCompletion(llmCtx, llm.ChatRequest{
		Messages: llmMessages,
	})
	if err != nil {
		s.log.Error("llm call failed", logger.Err(err))
		return session, canopyerr.Wrap(err, op)
	}

	// Append the assistant response.
	assistantMsg := domain.Message{Role: "assistant", Content: resp.Content}
	if err := session.AddMessage(assistantMsg); err != nil {
		return domain.Session{}, canopyerr.Wrap(err, op)
	}

	if err := s.sessions.Update(llmCtx, session); err != nil {
		return domain.Session{}, canopyerr.Wrap(err, op)
	}

	s.log.Debug("llm response added",
		logger.String("session_id", sessionID.String()),
		logger.Int("input_tokens", resp.Usage.InputTokens),
		logger.Int("output_tokens", resp.Usage.OutputTokens),
	)

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
