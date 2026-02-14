package postgres

import (
	"context"

	"github.com/0xsj/canopy-backend/internal/workspace/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/workspace/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// LLMConfigRepository implements domain.LLMConfigRepository using Postgres via sqlc.
type LLMConfigRepository struct {
	q *sqlc.Queries
}

// NewLLMConfigRepository creates a new LLMConfigRepository.
func NewLLMConfigRepository(db database.DBTX) *LLMConfigRepository {
	return &LLMConfigRepository{q: sqlc.New(db)}
}

var _ domain.LLMConfigRepository = (*LLMConfigRepository)(nil)

func (r *LLMConfigRepository) Upsert(ctx context.Context, cfg domain.LLMConfig) error {
	const op = "workspace: upsert llm config"
	return database.MapQueryError(r.q.UpsertLLMConfig(ctx, llmConfigToUpsertParams(cfg)), op)
}

func (r *LLMConfigRepository) FindByWorkspace(ctx context.Context, wsID types.WorkspaceID) (domain.LLMConfig, error) {
	const op = "workspace: find llm config by workspace"
	row, err := r.q.FindLLMConfigByWorkspace(ctx, wsID.String())
	if err != nil {
		return domain.LLMConfig{}, database.MapQueryError(err, op)
	}
	return llmConfigToDomain(row), nil
}

func (r *LLMConfigRepository) Delete(ctx context.Context, wsID types.WorkspaceID) error {
	const op = "workspace: delete llm config"
	tag, err := r.q.DeleteLLMConfig(ctx, wsID.String())
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}
