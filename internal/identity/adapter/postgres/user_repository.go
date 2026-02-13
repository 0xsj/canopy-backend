package postgres

import (
	"context"
	"time"

	"github.com/0xsj/canopy-backend/internal/identity/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// UserRepository implements domain.UserRepository using Postgres.
type UserRepository struct {
	db database.DBTX
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db database.DBTX) *UserRepository {
	return &UserRepository{db: db}
}

var _ domain.UserRepository = (*UserRepository)(nil)

func (r *UserRepository) Create(ctx context.Context, user domain.User) error {
	const op = "identity: create user"
	const query = `
		INSERT INTO users (id, external_id, display_name, email, avatar_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	ts := user.Timestamps()
	_, err := r.db.Exec(ctx, query,
		user.ID().String(),
		user.ExternalID(),
		user.DisplayName(),
		user.Email(),
		nullableString(user.AvatarURL()),
		ts.CreatedAt.Time(),
		tsUpdatedAt(ts),
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return nil
}

func (r *UserRepository) FindByID(ctx context.Context, id types.UserID) (domain.User, error) {
	const op = "identity: find user by id"
	const query = `
		SELECT id, external_id, display_name, email, avatar_url, created_at, updated_at
		FROM users WHERE id = $1`

	return scanUser(r.db.QueryRow(ctx, query, id.String()), op)
}

func (r *UserRepository) FindByExternalID(ctx context.Context, externalID string) (domain.User, error) {
	const op = "identity: find user by external id"
	const query = `
		SELECT id, external_id, display_name, email, avatar_url, created_at, updated_at
		FROM users WHERE external_id = $1`

	return scanUser(r.db.QueryRow(ctx, query, externalID), op)
}

func (r *UserRepository) FindByIDs(ctx context.Context, ids []types.UserID) ([]domain.User, error) {
	const op = "identity: find users by ids"
	const query = `
		SELECT id, external_id, display_name, email, avatar_url, created_at, updated_at
		FROM users WHERE id = ANY($1)`

	rows, err := r.db.Query(ctx, query, database.StringsFromIDs(ids))
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		u, err := scanUserRow(rows, op)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepository) Update(ctx context.Context, user domain.User) error {
	const op = "identity: update user"
	const query = `
		UPDATE users
		SET display_name = $2, email = $3, avatar_url = $4, updated_at = $5
		WHERE id = $1`

	ts := user.Timestamps()
	tag, err := r.db.Exec(ctx, query,
		user.ID().String(),
		user.DisplayName(),
		user.Email(),
		nullableString(user.AvatarURL()),
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

func scanUser(row rowScanner, op string) (domain.User, error) {
	var (
		rawID       string
		externalID  string
		displayName string
		email       string
		avatarURL   *string
		createdAt   time.Time
		updatedAt   time.Time
	)

	if err := row.Scan(&rawID, &externalID, &displayName, &email, &avatarURL, &createdAt, &updatedAt); err != nil {
		return domain.User{}, database.MapQueryError(err, op)
	}

	updatedTS := types.TimestampFrom(updatedAt)
	return domain.ReconstructUser(
		types.UserIDFrom(rawID),
		externalID,
		displayName,
		email,
		derefString(avatarURL),
		types.Timestamps{
			CreatedAt: types.TimestampFrom(createdAt),
			UpdatedAt: &updatedTS,
		},
	), nil
}

func scanUserRow(rows interface {
	Scan(dest ...any) error
}, op string) (domain.User, error) {
	return scanUser(rows, op)
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
