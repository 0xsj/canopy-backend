package postgres

import (
	"context"

	"github.com/0xsj/canopy-backend/internal/session/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/session/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// SessionRepository implements domain.SessionRepository using Postgres via sqlc.
type SessionRepository struct {
	q *sqlc.Queries
}

// NewSessionRepository creates a new SessionRepository.
func NewSessionRepository(db database.DBTX) *SessionRepository {
	return &SessionRepository{q: sqlc.New(db)}
}

var _ domain.SessionRepository = (*SessionRepository)(nil)

func (r *SessionRepository) Create(ctx context.Context, session domain.Session) error {
	const op = "session: create session"
	params, err := sessionToCreateParams(session)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return database.MapQueryError(r.q.CreateSession(ctx, params), op)
}

func (r *SessionRepository) FindByID(ctx context.Context, id domain.SessionID) (domain.Session, error) {
	const op = "session: find session by id"
	row, err := r.q.FindSessionByID(ctx, id.String())
	if err != nil {
		return domain.Session{}, database.MapQueryError(err, op)
	}
	s, err := sessionToDomain(row)
	if err != nil {
		return domain.Session{}, database.MapQueryError(err, op)
	}
	return s, nil
}

func (r *SessionRepository) FindActiveByUser(ctx context.Context, workspaceID types.WorkspaceID, userID types.UserID) ([]domain.Session, error) {
	const op = "session: find active sessions by user"
	rows, err := r.q.FindActiveSessionsByUser(ctx, sqlc.FindActiveSessionsByUserParams{
		WorkspaceID: workspaceID.String(),
		UserID:      userID.String(),
	})
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	sessions, err := sessionsToDomain(rows)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return sessions, nil
}

func (r *SessionRepository) Update(ctx context.Context, session domain.Session) error {
	const op = "session: update session"
	params, err := sessionToUpdateParams(session)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	tag, err := r.q.UpdateSession(ctx, params)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}
