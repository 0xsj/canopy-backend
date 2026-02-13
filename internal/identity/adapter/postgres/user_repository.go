package postgres

import (
	"context"

	"github.com/0xsj/canopy-backend/internal/identity/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/identity/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// UserRepository implements domain.UserRepository using Postgres via sqlc.
type UserRepository struct {
	q *sqlc.Queries
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db database.DBTX) *UserRepository {
	return &UserRepository{q: sqlc.New(db)}
}

var _ domain.UserRepository = (*UserRepository)(nil)

func (r *UserRepository) Create(ctx context.Context, user domain.User) error {
	const op = "identity: create user"
	return database.MapQueryError(r.q.CreateUser(ctx, userToCreateParams(user)), op)
}

func (r *UserRepository) FindByID(ctx context.Context, id types.UserID) (domain.User, error) {
	const op = "identity: find user by id"
	row, err := r.q.FindUserByID(ctx, id.String())
	if err != nil {
		return domain.User{}, database.MapQueryError(err, op)
	}
	return userToDomain(row), nil
}

func (r *UserRepository) FindByExternalID(ctx context.Context, externalID string) (domain.User, error) {
	const op = "identity: find user by external id"
	row, err := r.q.FindUserByExternalID(ctx, externalID)
	if err != nil {
		return domain.User{}, database.MapQueryError(err, op)
	}
	return userToDomain(row), nil
}

func (r *UserRepository) FindByIDs(ctx context.Context, ids []types.UserID) ([]domain.User, error) {
	const op = "identity: find users by ids"
	rows, err := r.q.FindUsersByIDs(ctx, database.StringsFromIDs(ids))
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return usersToDomain(rows), nil
}

func (r *UserRepository) Update(ctx context.Context, user domain.User) error {
	const op = "identity: update user"
	tag, err := r.q.UpdateUser(ctx, userToUpdateParams(user))
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}
