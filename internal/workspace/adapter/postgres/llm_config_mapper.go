package postgres

import (
	"github.com/0xsj/canopy-backend/internal/workspace/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/workspace/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// --- llm config: domain → sqlc params ---

func llmConfigToUpsertParams(cfg domain.LLMConfig) sqlc.UpsertLLMConfigParams {
	ts := cfg.Timestamps()
	return sqlc.UpsertLLMConfigParams{
		WorkspaceID: cfg.WorkspaceID().String(),
		Provider:    string(cfg.Provider()),
		Model:       cfg.Model(),
		ApiKeyEnc:   cfg.APIKeyEnc(),
		CreatedAt:   ts.CreatedAt.Time(),
		UpdatedAt:   database.TsUpdatedAt(ts),
	}
}

// --- llm config: sqlc model → domain ---

func llmConfigToDomain(row sqlc.LlmConfig) domain.LLMConfig {
	updatedTS := types.TimestampFrom(row.UpdatedAt)
	return domain.ReconstructLLMConfig(
		types.WorkspaceIDFrom(row.WorkspaceID),
		row.Provider,
		row.Model,
		row.ApiKeyEnc,
		types.Timestamps{
			CreatedAt: types.TimestampFrom(row.CreatedAt),
			UpdatedAt: &updatedTS,
		},
	)
}
