package postgres

import (
	"github.com/0xsj/canopy-backend/internal/deliverable/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/deliverable/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// --- domain → sqlc params ---

func deliverableToCreateParams(del domain.Deliverable) sqlc.CreateDeliverableParams {
	ts := del.Timestamps()
	return sqlc.CreateDeliverableParams{
		ID:            del.ID().String(),
		WorkspaceID:   del.WorkspaceID().String(),
		Format:        string(del.Format()),
		Content:       del.Content(),
		SourceLeafIds: database.StringsFromIDs(del.SourceLeafIDs()),
		Version:       int32(del.Version()),
		CreatedAt:     ts.CreatedAt.Time(),
		UpdatedAt:     database.TsUpdatedAt(ts),
		Finalized:     del.Finalized(),
	}
}

func deliverableToUpdateParams(del domain.Deliverable) sqlc.UpdateDeliverableParams {
	ts := del.Timestamps()
	return sqlc.UpdateDeliverableParams{
		ID:            del.ID().String(),
		Format:        string(del.Format()),
		Content:       del.Content(),
		SourceLeafIds: database.StringsFromIDs(del.SourceLeafIDs()),
		Version:       int32(del.Version()),
		UpdatedAt:     database.TsUpdatedAt(ts),
		Finalized:     del.Finalized(),
	}
}

// --- sqlc model → domain ---

func deliverableToDomain(row sqlc.Deliverable) domain.Deliverable {
	sourceLeafIDs := make([]types.LeafID, len(row.SourceLeafIds))
	for i, s := range row.SourceLeafIds {
		sourceLeafIDs[i] = types.LeafIDFrom(s)
	}
	updatedTS := types.TimestampFrom(row.UpdatedAt)
	return domain.ReconstructDeliverable(
		types.DeliverableIDFrom(row.ID),
		types.WorkspaceIDFrom(row.WorkspaceID),
		domain.Format(row.Format),
		row.Content,
		sourceLeafIDs,
		int(row.Version),
		row.Finalized,
		types.Timestamps{
			CreatedAt: types.TimestampFrom(row.CreatedAt),
			UpdatedAt: &updatedTS,
		},
	)
}

func deliverablesToDomain(rows []sqlc.Deliverable) []domain.Deliverable {
	dels := make([]domain.Deliverable, len(rows))
	for i, row := range rows {
		dels[i] = deliverableToDomain(row)
	}
	return dels
}
