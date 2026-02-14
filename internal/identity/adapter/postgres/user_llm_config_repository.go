package postgres

import (
	"context"

	"github.com/0xsj/canopy-backend/internal/identity/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/identity/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// UserLLMConfigRepository implements domain.UserLLMConfigRepository using Postgres via sqlc.
type UserLLMConfigRepository struct {
	q *sqlc.Queries
}

// NewUserLLMConfigRepository creates a new UserLLMConfigRepository.
func NewUserLLMConfigRepository(db database.DBTX) *UserLLMConfigRepository {
	return &UserLLMConfigRepository{q: sqlc.New(db)}
}

var _ domain.UserLLMConfigRepository = (*UserLLMConfigRepository)(nil)

func (r *UserLLMConfigRepository) Upsert(ctx context.Context, cfg domain.UserLLMConfig) error {
	const op = "identity: upsert user llm config"
	return database.MapQueryError(r.q.UpsertUserLLMConfig(ctx, userLLMConfigToUpsertParams(cfg)), op)
}

func (r *UserLLMConfigRepository) FindByUser(ctx context.Context, userID types.UserID) (domain.UserLLMConfig, error) {
	const op = "identity: find user llm config"
	row, err := r.q.FindUserLLMConfigByUser(ctx, userID.String())
	if err != nil {
		return domain.UserLLMConfig{}, database.MapQueryError(err, op)
	}
	return userLLMConfigToDomain(row), nil
}

func (r *UserLLMConfigRepository) Delete(ctx context.Context, userID types.UserID) error {
	const op = "identity: delete user llm config"
	tag, err := r.q.DeleteUserLLMConfig(ctx, userID.String())
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}
