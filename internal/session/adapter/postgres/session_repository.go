package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/0xsj/canopy-backend/internal/session/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// SessionRepository implements domain.SessionRepository using Postgres.
type SessionRepository struct {
	db database.DBTX
}

// NewSessionRepository creates a new SessionRepository.
func NewSessionRepository(db database.DBTX) *SessionRepository {
	return &SessionRepository{db: db}
}

var _ domain.SessionRepository = (*SessionRepository)(nil)

func (r *SessionRepository) Create(ctx context.Context, session domain.Session) error {
	const op = "session: create session"
	const query = `
		INSERT INTO sessions (id, workspace_id, user_id, seed_id, parent_leaf_id,
			session_type, status, messages, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	messagesJSON, err := json.Marshal(session.Messages())
	if err != nil {
		return database.MapQueryError(err, op)
	}

	ts := session.Timestamps()
	_, err = r.db.Exec(ctx, query,
		session.ID().String(),
		session.WorkspaceID().String(),
		session.UserID().String(),
		session.SeedID().String(),
		nullableString(session.ParentLeafID().String()),
		string(session.Type()),
		string(session.Status()),
		messagesJSON,
		ts.CreatedAt.Time(),
		tsUpdatedAt(ts),
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return nil
}

func (r *SessionRepository) FindByID(ctx context.Context, id domain.SessionID) (domain.Session, error) {
	const op = "session: find session by id"
	const query = sessionSelectColumns + ` FROM sessions WHERE id = $1`

	return scanSession(r.db.QueryRow(ctx, query, id.String()), op)
}

func (r *SessionRepository) FindActiveByUser(ctx context.Context, workspaceID types.WorkspaceID, userID types.UserID) ([]domain.Session, error) {
	const op = "session: find active sessions by user"
	const query = sessionSelectColumns + `
		FROM sessions
		WHERE workspace_id = $1 AND user_id = $2 AND status NOT IN ('completed', 'abandoned')
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, workspaceID.String(), userID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	defer rows.Close()

	var sessions []domain.Session
	for rows.Next() {
		s, err := scanSession(rows, op)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *SessionRepository) Update(ctx context.Context, session domain.Session) error {
	const op = "session: update session"
	const query = `
		UPDATE sessions
		SET status = $2, messages = $3, updated_at = $4
		WHERE id = $1`

	messagesJSON, err := json.Marshal(session.Messages())
	if err != nil {
		return database.MapQueryError(err, op)
	}

	ts := session.Timestamps()
	tag, err := r.db.Exec(ctx, query,
		session.ID().String(),
		string(session.Status()),
		messagesJSON,
		tsUpdatedAt(ts),
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}

// --- helpers ---

type rowScanner interface {
	Scan(dest ...any) error
}

const sessionSelectColumns = `SELECT id, workspace_id, user_id, seed_id, parent_leaf_id,
	session_type, status, messages, created_at, updated_at`

func scanSession(row rowScanner, op string) (domain.Session, error) {
	var (
		rawID         string
		rawWorkspace  string
		rawUser       string
		rawSeed       string
		rawParentLeaf *string
		sessionType   string
		status        string
		messagesJSON  []byte
		createdAt     time.Time
		updatedAt     time.Time
	)

	if err := row.Scan(&rawID, &rawWorkspace, &rawUser, &rawSeed, &rawParentLeaf,
		&sessionType, &status, &messagesJSON, &createdAt, &updatedAt); err != nil {
		return domain.Session{}, database.MapQueryError(err, op)
	}

	var messages []domain.Message
	if len(messagesJSON) > 0 {
		if err := json.Unmarshal(messagesJSON, &messages); err != nil {
			return domain.Session{}, database.MapQueryError(err, op)
		}
	}

	updatedTS := types.TimestampFrom(updatedAt)
	return domain.ReconstructSession(
		domain.SessionIDFrom(rawID),
		types.WorkspaceIDFrom(rawWorkspace),
		types.UserIDFrom(rawUser),
		types.SeedIDFrom(rawSeed),
		types.LeafIDFrom(derefString(rawParentLeaf)),
		domain.SessionType(sessionType),
		domain.SessionStatus(status),
		messages,
		types.Timestamps{
			CreatedAt: types.TimestampFrom(createdAt),
			UpdatedAt: &updatedTS,
		},
	), nil
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func tsUpdatedAt(ts types.Timestamps) time.Time {
	if ts.UpdatedAt != nil {
		return ts.UpdatedAt.Time()
	}
	return ts.CreatedAt.Time()
}
