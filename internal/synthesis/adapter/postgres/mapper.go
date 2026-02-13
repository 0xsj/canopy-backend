package postgres

import (
	"github.com/0xsj/canopy-backend/internal/synthesis/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/synthesis/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// --- domain → sqlc params ---

func workflowToCreateParams(w domain.SynthesisWorkflow) sqlc.CreateWorkflowParams {
	ts := w.Timestamps()
	return sqlc.CreateWorkflowParams{
		ID:            w.ID().String(),
		WorkspaceID:   w.WorkspaceID().String(),
		InitiatorID:   w.InitiatorID().String(),
		SourceLeafIds: database.StringsFromIDs(w.SourceLeafIDs()),
		ResultLeafID:  database.NullableString(w.ResultLeafID().String()),
		Status:        string(w.Status()),
		FailureReason: database.NullableString(w.FailureReason()),
		CreatedAt:     ts.CreatedAt.Time(),
		UpdatedAt:     database.TsUpdatedAt(ts),
	}
}

func workflowToUpdateParams(w domain.SynthesisWorkflow) sqlc.UpdateWorkflowParams {
	ts := w.Timestamps()
	return sqlc.UpdateWorkflowParams{
		ID:            w.ID().String(),
		Status:        string(w.Status()),
		ResultLeafID:  database.NullableString(w.ResultLeafID().String()),
		FailureReason: database.NullableString(w.FailureReason()),
		UpdatedAt:     database.TsUpdatedAt(ts),
	}
}

// --- sqlc model → domain ---

func workflowToDomain(row sqlc.Workflow) domain.SynthesisWorkflow {
	sourceLeafIDs := make([]types.LeafID, len(row.SourceLeafIds))
	for i, s := range row.SourceLeafIds {
		sourceLeafIDs[i] = types.LeafIDFrom(s)
	}
	updatedTS := types.TimestampFrom(row.UpdatedAt)
	return domain.ReconstructSynthesisWorkflow(
		domain.SynthesisIDFrom(row.ID),
		types.WorkspaceIDFrom(row.WorkspaceID),
		types.UserIDFrom(row.InitiatorID),
		sourceLeafIDs,
		types.LeafIDFrom(database.DerefString(row.ResultLeafID)),
		domain.WorkflowStatus(row.Status),
		database.DerefString(row.FailureReason),
		types.Timestamps{
			CreatedAt: types.TimestampFrom(row.CreatedAt),
			UpdatedAt: &updatedTS,
		},
	)
}

func workflowsToDomain(rows []sqlc.Workflow) []domain.SynthesisWorkflow {
	workflows := make([]domain.SynthesisWorkflow, len(rows))
	for i, row := range rows {
		workflows[i] = workflowToDomain(row)
	}
	return workflows
}
