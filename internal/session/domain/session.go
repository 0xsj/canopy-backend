package domain

import (
	"fmt"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// SessionType determines the kind of AI-assisted session.
type SessionType string

const (
	SessionExploration SessionType = "exploration"
	SessionShaping     SessionType = "shaping"
	SessionSynthesis   SessionType = "synthesis"
	SessionDigest      SessionType = "digest"
)

// IsValid reports whether the session type is a recognized value.
func (s SessionType) IsValid() bool {
	switch s {
	case SessionExploration, SessionShaping, SessionSynthesis, SessionDigest:
		return true
	}
	return false
}

// SessionStatus tracks the lifecycle of a session.
type SessionStatus string

const (
	StatusActive     SessionStatus = "active"
	StatusCheckpoint SessionStatus = "checkpoint" // shaping phase entered
	StatusCompleted  SessionStatus = "completed"
	StatusAbandoned  SessionStatus = "abandoned"
)

// IsValid reports whether the status is a recognized value.
func (s SessionStatus) IsValid() bool {
	switch s {
	case StatusActive, StatusCheckpoint, StatusCompleted, StatusAbandoned:
		return true
	}
	return false
}

// IsTerminal reports whether the session is in a final state.
func (s SessionStatus) IsTerminal() bool {
	return s == StatusCompleted || s == StatusAbandoned
}

// Message represents a single exchange in a session conversation.
type Message struct {
	Role    string `json:"role"` // "system", "user", "assistant"
	Content string `json:"content"`
}

// Session manages the lifecycle of an AI-assisted prompting session.
// Sessions have two phases: exploration (open conversation) and shaping
// (refining toward a structured leaf). The session tracks the full
// conversation history and produces a leaf on completion.
type Session struct {
	id           types.ID[sessionTag]
	workspaceID  types.WorkspaceID
	userID       types.UserID
	seedID       types.SeedID
	parentLeafID types.LeafID // zero if exploring from seed directly
	sessionType  SessionType
	status       SessionStatus
	messages     []Message
	timestamps   types.Timestamps
}

type sessionTag struct{}

// SessionID is the exported type alias for use in adapter packages.
type SessionID = types.ID[sessionTag]

const prefixSession = "ses"

// NewSession creates a new active session.
func NewSession(
	workspaceID types.WorkspaceID,
	userID types.UserID,
	seedID types.SeedID,
	parentLeafID types.LeafID,
	sessionType SessionType,
) (Session, error) {
	if workspaceID.IsZero() {
		return Session{}, fmt.Errorf("session: workspace ID is required")
	}
	if userID.IsZero() {
		return Session{}, fmt.Errorf("session: user ID is required")
	}
	if seedID.IsZero() {
		return Session{}, fmt.Errorf("session: seed ID is required")
	}
	if !sessionType.IsValid() {
		return Session{}, fmt.Errorf("session: invalid session type %q", sessionType)
	}

	return Session{
		id:           types.NewID[sessionTag](prefixSession),
		workspaceID:  workspaceID,
		userID:       userID,
		seedID:       seedID,
		parentLeafID: parentLeafID,
		sessionType:  sessionType,
		status:       StatusActive,
		messages:     nil,
		timestamps:   types.NewMutableTimestamps(),
	}, nil
}

// ReconstructSession builds a Session from trusted data.
func ReconstructSession(
	id types.ID[sessionTag],
	workspaceID types.WorkspaceID,
	userID types.UserID,
	seedID types.SeedID,
	parentLeafID types.LeafID,
	sessionType SessionType,
	status SessionStatus,
	messages []Message,
	timestamps types.Timestamps,
) Session {
	return Session{
		id:           id,
		workspaceID:  workspaceID,
		userID:       userID,
		seedID:       seedID,
		parentLeafID: parentLeafID,
		sessionType:  sessionType,
		status:       status,
		messages:     messages,
		timestamps:   timestamps,
	}
}

// AddMessage appends a message to the conversation. Only active or
// checkpoint sessions accept messages.
func (s *Session) AddMessage(msg Message) error {
	if s.status.IsTerminal() {
		return fmt.Errorf("session: cannot add message to %s session", s.status)
	}
	s.messages = append(s.messages, msg)
	s.timestamps.Touch()
	return nil
}

// EnterCheckpoint transitions the session from active to the shaping phase.
func (s *Session) EnterCheckpoint() error {
	if s.status != StatusActive {
		return fmt.Errorf("session: can only checkpoint an active session, current status: %s", s.status)
	}
	s.status = StatusCheckpoint
	s.timestamps.Touch()
	return nil
}

// Complete marks the session as successfully completed (leaf was confirmed).
func (s *Session) Complete() error {
	if s.status.IsTerminal() {
		return fmt.Errorf("session: session already %s", s.status)
	}
	s.status = StatusCompleted
	s.timestamps.Touch()
	return nil
}

// Abandon marks the session as abandoned (user cancelled).
func (s *Session) Abandon() error {
	if s.status.IsTerminal() {
		return fmt.Errorf("session: session already %s", s.status)
	}
	s.status = StatusAbandoned
	s.timestamps.Touch()
	return nil
}

func (s Session) ID() types.ID[sessionTag]       { return s.id }
func (s Session) WorkspaceID() types.WorkspaceID { return s.workspaceID }
func (s Session) UserID() types.UserID           { return s.userID }
func (s Session) SeedID() types.SeedID           { return s.seedID }
func (s Session) ParentLeafID() types.LeafID     { return s.parentLeafID }
func (s Session) Type() SessionType              { return s.sessionType }
func (s Session) Status() SessionStatus          { return s.status }
func (s Session) Messages() []Message            { return s.messages }
func (s Session) Timestamps() types.Timestamps   { return s.timestamps }

// SessionIDFrom creates a SessionID from a trusted database string.
func SessionIDFrom(raw string) types.ID[sessionTag] { return types.IDFrom[sessionTag](raw) }

// ParseSessionID validates and parses an untrusted session ID string.
func ParseSessionID(raw string) (SessionID, error) {
	return types.ParseID[sessionTag](raw, prefixSession)
}
