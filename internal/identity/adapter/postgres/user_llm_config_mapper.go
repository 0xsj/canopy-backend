package postgres

import (
	"github.com/0xsj/canopy-backend/internal/identity/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/identity/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// --- user llm config: domain → sqlc params ---

func userLLMConfigToUpsertParams(cfg domain.UserLLMConfig) sqlc.UpsertUserLLMConfigParams {
	ts := cfg.Timestamps()
	return sqlc.UpsertUserLLMConfigParams{
		UserID:    cfg.UserID().String(),
		Provider:  string(cfg.Provider()),
		Model:     cfg.Model(),
		ApiKeyEnc: cfg.APIKeyEnc(),
		CreatedAt: ts.CreatedAt.Time(),
		UpdatedAt: database.TsUpdatedAt(ts),
	}
}

// --- user llm config: sqlc model → domain ---

func userLLMConfigToDomain(row sqlc.UserLlmConfig) domain.UserLLMConfig {
	updatedTS := types.TimestampFrom(row.UpdatedAt)
	return domain.ReconstructUserLLMConfig(
		types.UserIDFrom(row.UserID),
		row.Provider,
		row.Model,
		row.ApiKeyEnc,
		types.Timestamps{
			CreatedAt: types.TimestampFrom(row.CreatedAt),
			UpdatedAt: &updatedTS,
		},
	)
}
