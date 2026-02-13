package postgres

import (
	"context"
	"time"

	"github.com/0xsj/canopy-backend/internal/convergence/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// SignalRepository implements domain.SignalRepository using Postgres.
type SignalRepository struct {
	db database.DBTX
}

// NewSignalRepository creates a new SignalRepository.
func NewSignalRepository(db database.DBTX) *SignalRepository {
	return &SignalRepository{db: db}
}

var _ domain.SignalRepository = (*SignalRepository)(nil)

func (r *SignalRepository) Create(ctx context.Context, signal domain.Signal) error {
	const op = "convergence: create signal"
	const query = `
		INSERT INTO signals (id, workspace_id, leaf_id, user_id, signal_type, annotation, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.db.Exec(ctx, query,
		signal.ID().String(),
		signal.WorkspaceID().String(),
		signal.LeafID().String(),
		signal.UserID().String(),
		string(signal.Type()),
		nullableString(signal.Annotation()),
		signal.Timestamps().CreatedAt.Time(),
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return nil
}

func (r *SignalRepository) Delete(ctx context.Context, id domain.SignalID) error {
	const op = "convergence: delete signal"
	const query = `DELETE FROM signals WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, id.String())
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}

func (r *SignalRepository) FindByLeaf(ctx context.Context, leafID types.LeafID) ([]domain.Signal, error) {
	const op = "convergence: find signals by leaf"
	const query = signalSelectColumns + ` FROM signals WHERE leaf_id = $1 ORDER BY created_at`

	return r.querySignals(ctx, query, op, leafID.String())
}

func (r *SignalRepository) FindByUser(ctx context.Context, workspaceID types.WorkspaceID, userID types.UserID) ([]domain.Signal, error) {
	const op = "convergence: find signals by user"
	const query = signalSelectColumns + ` FROM signals WHERE workspace_id = $1 AND user_id = $2 ORDER BY created_at`

	return r.querySignals(ctx, query, op, workspaceID.String(), userID.String())
}

func (r *SignalRepository) FindByLeafAndUser(ctx context.Context, leafID types.LeafID, userID types.UserID) (domain.Signal, error) {
	const op = "convergence: find signal by leaf and user"
	const query = signalSelectColumns + ` FROM signals WHERE leaf_id = $1 AND user_id = $2`

	return scanSignal(r.db.QueryRow(ctx, query, leafID.String(), userID.String()), op)
}

func (r *SignalRepository) CountByLeaf(ctx context.Context, leafID types.LeafID) (map[domain.SignalType]int, error) {
	const op = "convergence: count signals by leaf"
	const query = `
		SELECT signal_type, COUNT(*)
		FROM signals WHERE leaf_id = $1
		GROUP BY signal_type`

	rows, err := r.db.Query(ctx, query, leafID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	defer rows.Close()

	counts := make(map[domain.SignalType]int)
	for rows.Next() {
		var signalType string
		var count int
		if err := rows.Scan(&signalType, &count); err != nil {
			return nil, database.MapQueryError(err, op)
		}
		counts[domain.SignalType(signalType)] = count
	}
	return counts, rows.Err()
}

// --- helpers ---

type rowScanner interface {
	Scan(dest ...any) error
}

const signalSelectColumns = `SELECT id, workspace_id, leaf_id, user_id, signal_type, annotation, created_at`

func (r *SignalRepository) querySignals(ctx context.Context, query, op string, args ...any) ([]domain.Signal, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	defer rows.Close()

	var signals []domain.Signal
	for rows.Next() {
		s, err := scanSignal(rows, op)
		if err != nil {
			return nil, err
		}
		signals = append(signals, s)
	}
	return signals, rows.Err()
}

func scanSignal(row rowScanner, op string) (domain.Signal, error) {
	var (
		rawID        string
		rawWorkspace string
		rawLeaf      string
		rawUser      string
		signalType   string
		annotation   *string
		createdAt    time.Time
	)

	if err := row.Scan(&rawID, &rawWorkspace, &rawLeaf, &rawUser, &signalType, &annotation, &createdAt); err != nil {
		return domain.Signal{}, database.MapQueryError(err, op)
	}

	return domain.ReconstructSignal(
		domain.SignalIDFrom(rawID),
		types.WorkspaceIDFrom(rawWorkspace),
		types.LeafIDFrom(rawLeaf),
		types.UserIDFrom(rawUser),
		domain.SignalType(signalType),
		derefString(annotation),
		types.Timestamps{CreatedAt: types.TimestampFrom(createdAt)},
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
