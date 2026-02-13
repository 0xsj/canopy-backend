package postgres

import (
	"encoding/json"

	"github.com/0xsj/canopy-backend/internal/seed/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/seed/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// --- domain → sqlc params ---

func seedToCreateParams(s domain.Seed) (sqlc.CreateSeedParams, error) {
	constraintsJSON, err := json.Marshal(s.Constraints())
	if err != nil {
		return sqlc.CreateSeedParams{}, err
	}
	ts := s.Timestamps()
	return sqlc.CreateSeedParams{
		ID:          s.ID().String(),
		WorkspaceID: s.WorkspaceID().String(),
		AuthorID:    s.AuthorID().String(),
		Title:       s.Title(),
		Description: database.NullableString(s.Description()),
		Constraints: constraintsJSON,
		Tags:        s.Tags(),
		CreatedAt:   ts.CreatedAt.Time(),
		UpdatedAt:   database.TsUpdatedAt(ts),
	}, nil
}

func seedToUpdateParams(s domain.Seed) (sqlc.UpdateSeedParams, error) {
	constraintsJSON, err := json.Marshal(s.Constraints())
	if err != nil {
		return sqlc.UpdateSeedParams{}, err
	}
	ts := s.Timestamps()
	return sqlc.UpdateSeedParams{
		ID:          s.ID().String(),
		Constraints: constraintsJSON,
		Tags:        s.Tags(),
		UpdatedAt:   database.TsUpdatedAt(ts),
	}, nil
}

// --- sqlc model → domain ---

func seedToDomain(row sqlc.Seed) (domain.Seed, error) {
	constraints := make(map[string]any)
	if len(row.Constraints) > 0 {
		if err := json.Unmarshal(row.Constraints, &constraints); err != nil {
			return domain.Seed{}, err
		}
	}
	updatedTS := types.TimestampFrom(row.UpdatedAt)
	return domain.ReconstructSeed(
		types.SeedIDFrom(row.ID),
		types.WorkspaceIDFrom(row.WorkspaceID),
		types.UserIDFrom(row.AuthorID),
		row.Title,
		database.DerefString(row.Description),
		constraints,
		row.Tags,
		types.Timestamps{
			CreatedAt: types.TimestampFrom(row.CreatedAt),
			UpdatedAt: &updatedTS,
		},
	), nil
}

func seedsToDomain(rows []sqlc.Seed) ([]domain.Seed, error) {
	seeds := make([]domain.Seed, len(rows))
	for i, row := range rows {
		s, err := seedToDomain(row)
		if err != nil {
			return nil, err
		}
		seeds[i] = s
	}
	return seeds, nil
}
