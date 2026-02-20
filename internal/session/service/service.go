package service

import (
	"context"
	"strings"
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
	sessions    domain.SessionRepository
	assembler   domain.ContextAssembler
	llm         llm.ProviderResolver
	broadcaster llm.StreamBroadcaster
	wsMembers   WorkspaceMemberReader
	pub         events.Publisher
	log         logger.Logger
}

// New creates a new session service.
func New(
	sessions domain.SessionRepository,
	assembler domain.ContextAssembler,
	llmResolver llm.ProviderResolver,
	broadcaster llm.StreamBroadcaster,
	wsMembers WorkspaceMemberReader,
	pub events.Publisher,
	log logger.Logger,
) *Service {
	return &Service{
		sessions:    sessions,
		assembler:   assembler,
		llm:         llmResolver,
		broadcaster: broadcaster,
		wsMembers:   wsMembers,
		pub:         pub,
		log:         log,
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

// AddMessageStreaming appends a user message synchronously, then spawns a background
// goroutine to stream the LLM response. Chunks are delivered via the StreamBroadcaster.
// Returns the stream ID (session ID) immediately for the caller to track progress.
func (s *Service) AddMessageStreaming(ctx context.Context, sessionID domain.SessionID, msg domain.Message) (string, error) {
	const op = "session: add message streaming"

	session, err := s.sessions.FindByID(ctx, sessionID)
	if err != nil {
		return "", canopyerr.Wrap(err, op)
	}

	if _, err := s.requireMember(ctx, session.WorkspaceID()); err != nil {
		return "", canopyerr.Wrap(err, op)
	}

	// Idempotency guard (same as AddMessage).
	if msgs := session.Messages(); len(msgs) > 0 {
		for i := len(msgs) - 1; i >= 0 && i >= len(msgs)-2; i-- {
			if msgs[i].Role == msg.Role && msgs[i].Content == msg.Content {
				s.log.Info("duplicate message skipped (streaming)",
					logger.String("session_id", sessionID.String()),
					logger.String("role", msg.Role),
				)
				return sessionID.String(), nil
			}
		}
	}

	// Append the user message synchronously.
	if err := session.AddMessage(msg); err != nil {
		return "", canopyerr.Wrap(err, op)
	}
	if err := s.sessions.Update(ctx, session); err != nil {
		return "", canopyerr.Wrap(err, op)
	}

	streamID := sessionID.String()

	// Spawn background goroutine with detached context.
	go s.runStream(context.WithoutCancel(ctx), session, streamID)

	return streamID, nil
}

// runStream assembles context, resolves the LLM provider, and streams the response.
// It sends stream.start/chunk/end/error messages via the broadcaster.
func (s *Service) runStream(ctx context.Context, session domain.Session, streamID string) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	wsID := session.WorkspaceID().String()

	_ = s.broadcaster.Send(wsID, llm.StreamMessage{
		Type:     "stream.start",
		StreamID: streamID,
	})

	assembled, err := s.assembler.Assemble(ctx, session)
	if err != nil {
		s.log.Error("stream: context assembly failed", logger.Err(err))
		_ = s.broadcaster.Send(wsID, llm.StreamMessage{
			Type:     "stream.error",
			StreamID: streamID,
			Error:    "context assembly failed",
		})
		return
	}

	llmMessages := make([]llm.Message, len(assembled))
	for i, m := range assembled {
		llmMessages[i] = llm.Message{Role: m.Role, Content: m.Content}
	}

	provider, err := s.llm.Resolve(ctx, session.WorkspaceID())
	if err != nil {
		s.log.Error("stream: llm resolve failed", logger.Err(err))
		_ = s.broadcaster.Send(wsID, llm.StreamMessage{
			Type:     "stream.error",
			StreamID: streamID,
			Error:    "failed to resolve LLM provider",
		})
		return
	}

	req := llm.ChatRequest{Messages: llmMessages}

	// Try streaming; fall back to sync if provider doesn't support it.
	sp, ok := provider.(llm.StreamProvider)
	if !ok {
		s.runStreamFallback(ctx, provider, req, session, streamID, wsID)
		return
	}

	var accumulated strings.Builder

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
	if err != nil {
		s.log.Error("stream: llm stream failed", logger.Err(err))
		_ = s.broadcaster.Send(wsID, llm.StreamMessage{
			Type:     "stream.error",
			StreamID: streamID,
			Error:    "LLM streaming failed",
		})
		return
	}

	content := accumulated.String()
	s.persistAssistantMessage(ctx, session, content, streamID)

	_ = s.broadcaster.Send(wsID, llm.StreamMessage{
		Type:     "stream.end",
		StreamID: streamID,
		Content:  content,
	})
}

// runStreamFallback handles the case where the provider doesn't support streaming.
// It calls ChatCompletion synchronously and sends the full response as a single stream.end.
func (s *Service) runStreamFallback(ctx context.Context, provider llm.Provider, req llm.ChatRequest, session domain.Session, streamID, wsID string) {
	resp, err := provider.ChatCompletion(ctx, req)
	if err != nil {
		s.log.Error("stream fallback: llm call failed", logger.Err(err))
		_ = s.broadcaster.Send(wsID, llm.StreamMessage{
			Type:     "stream.error",
			StreamID: streamID,
			Error:    "LLM call failed",
		})
		return
	}

	s.persistAssistantMessage(ctx, session, resp.Content, streamID)

	_ = s.broadcaster.Send(wsID, llm.StreamMessage{
		Type:     "stream.end",
		StreamID: streamID,
		Content:  resp.Content,
	})
}

// persistAssistantMessage appends the assistant response to the session and persists it.
func (s *Service) persistAssistantMessage(ctx context.Context, session domain.Session, content, streamID string) {
	assistantMsg := domain.Message{Role: "assistant", Content: content}
	if err := session.AddMessage(assistantMsg); err != nil {
		s.log.Error("stream: failed to add assistant message",
			logger.String("stream_id", streamID),
			logger.Err(err),
		)
		return
	}
	if err := s.sessions.Update(ctx, session); err != nil {
		s.log.Error("stream: failed to persist assistant message",
			logger.String("stream_id", streamID),
			logger.Err(err),
		)
	}
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
