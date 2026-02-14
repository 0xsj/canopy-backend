package domain

import (
	"context"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// SessionRepository defines the persistence port for session entities.
type SessionRepository interface {
	// Create persists a new session.
	Create(ctx context.Context, session Session) error

	// FindByID returns a session by ID.
	FindByID(ctx context.Context, id types.ID[sessionTag]) (Session, error)

	// FindByWorkspace returns all sessions in a workspace, ordered by most recent first.
	FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]Session, error)

	// FindActiveByUser returns all active (non-terminal) sessions for a user
	// in a workspace.
	FindActiveByUser(ctx context.Context, workspaceID types.WorkspaceID, userID types.UserID) ([]Session, error)

	// Update persists changes to a session (new messages, status transitions).
	Update(ctx context.Context, session Session) error
}

// ContextAssembler defines the port for assembling LLM prompt context.
// Different session types assemble context differently — exploration pulls
// seed constraints and parent leaf content, synthesis pulls source leaves, etc.
type ContextAssembler interface {
	// Assemble builds the message context for an LLM call based on
	// the session type and current state.
	Assemble(ctx context.Context, session Session) ([]Message, error)
}
