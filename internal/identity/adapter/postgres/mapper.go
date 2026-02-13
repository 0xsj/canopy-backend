package postgres

import (
	"github.com/0xsj/canopy-backend/internal/identity/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/identity/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// --- domain → sqlc params ---

func userToCreateParams(u domain.User) sqlc.CreateUserParams {
	ts := u.Timestamps()
	return sqlc.CreateUserParams{
		ID:          u.ID().String(),
		ExternalID:  u.ExternalID(),
		DisplayName: u.DisplayName(),
		Email:       u.Email(),
		AvatarUrl:   database.NullableString(u.AvatarURL()),
		CreatedAt:   ts.CreatedAt.Time(),
		UpdatedAt:   database.TsUpdatedAt(ts),
	}
}

func userToUpdateParams(u domain.User) sqlc.UpdateUserParams {
	ts := u.Timestamps()
	return sqlc.UpdateUserParams{
		ID:          u.ID().String(),
		DisplayName: u.DisplayName(),
		Email:       u.Email(),
		AvatarUrl:   database.NullableString(u.AvatarURL()),
		UpdatedAt:   database.TsUpdatedAt(ts),
	}
}

// --- sqlc model → domain ---

func userToDomain(row sqlc.User) domain.User {
	updatedTS := types.TimestampFrom(row.UpdatedAt)
	return domain.ReconstructUser(
		types.UserIDFrom(row.ID),
		row.ExternalID,
		row.DisplayName,
		row.Email,
		database.DerefString(row.AvatarUrl),
		types.Timestamps{
			CreatedAt: types.TimestampFrom(row.CreatedAt),
			UpdatedAt: &updatedTS,
		},
	)
}

func usersToDomain(rows []sqlc.User) []domain.User {
	users := make([]domain.User, len(rows))
	for i, row := range rows {
		users[i] = userToDomain(row)
	}
	return users
}
