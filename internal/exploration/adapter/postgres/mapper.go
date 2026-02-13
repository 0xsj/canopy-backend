package postgres

import (
	"encoding/json"

	"github.com/0xsj/canopy-backend/internal/exploration/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/exploration/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// --- leaf: domain → sqlc params ---

func leafToCreateParams(l domain.Leaf) (sqlc.CreateLeafParams, error) {
	sourcesJSON, err := marshalSources(l.Sources())
	if err != nil {
		return sqlc.CreateLeafParams{}, err
	}
	metadataJSON, err := json.Marshal(l.Metadata())
	if err != nil {
		return sqlc.CreateLeafParams{}, err
	}
	return sqlc.CreateLeafParams{
		ID:            l.ID().String(),
		WorkspaceID:   l.WorkspaceID().String(),
		SeedID:        l.SeedID().String(),
		BranchID:      l.BranchID().String(),
		AuthorID:      l.AuthorID().String(),
		ParentLeafID:  database.NullableString(l.ParentLeafID().String()),
		Title:         l.Title(),
		Summary:       l.Summary(),
		KeyPoints:     l.KeyPoints(),
		OpenQuestions: l.OpenQuestions(),
		Tags:          l.Tags(),
		Layer:         string(l.Layer()),
		Sources:       sourcesJSON,
		Metadata:      metadataJSON,
		CreatedAt:     l.Timestamps().CreatedAt.Time(),
	}, nil
}

func leafFilterToParams(workspaceID types.WorkspaceID, f domain.LeafFilter) sqlc.FindLeavesByWorkspaceParams {
	p := sqlc.FindLeavesByWorkspaceParams{WorkspaceID: workspaceID.String()}
	if f.AuthorID != nil {
		s := f.AuthorID.String()
		p.AuthorID = &s
	}
	if f.SeedID != nil {
		s := f.SeedID.String()
		p.SeedID = &s
	}
	if f.Layer != nil {
		s := string(*f.Layer)
		p.Layer = &s
	}
	if len(f.Tags) > 0 {
		p.Tags = f.Tags
	}
	return p
}

// --- leaf: sqlc model → domain ---

func leafToDomain(row sqlc.Leafe) (domain.Leaf, error) {
	sources, err := unmarshalSources(row.Sources)
	if err != nil {
		return domain.Leaf{}, err
	}
	metadata := make(map[string]any)
	if len(row.Metadata) > 0 {
		if err := json.Unmarshal(row.Metadata, &metadata); err != nil {
			return domain.Leaf{}, err
		}
	}
	return domain.ReconstructLeaf(
		types.LeafIDFrom(row.ID),
		types.WorkspaceIDFrom(row.WorkspaceID),
		types.SeedIDFrom(row.SeedID),
		types.BranchIDFrom(row.BranchID),
		types.UserIDFrom(row.AuthorID),
		types.LeafIDFrom(database.DerefString(row.ParentLeafID)),
		row.Title,
		row.Summary,
		row.KeyPoints,
		row.OpenQuestions,
		row.Tags,
		domain.Layer(row.Layer),
		sources,
		metadata,
		types.Timestamps{CreatedAt: types.TimestampFrom(row.CreatedAt)},
	), nil
}

func leavesToDomain(rows []sqlc.Leafe) ([]domain.Leaf, error) {
	leaves := make([]domain.Leaf, len(rows))
	for i, row := range rows {
		l, err := leafToDomain(row)
		if err != nil {
			return nil, err
		}
		leaves[i] = l
	}
	return leaves, nil
}

// --- leaf sources JSONB ---

func marshalSources(sources []domain.Source) (json.RawMessage, error) {
	if sources == nil {
		return nil, nil
	}
	return json.Marshal(sources)
}

func unmarshalSources(data json.RawMessage) ([]domain.Source, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var sources []domain.Source
	if err := json.Unmarshal(data, &sources); err != nil {
		return nil, err
	}
	return sources, nil
}

// --- branch: domain → sqlc params ---

func branchToCreateParams(b domain.Branch) sqlc.CreateBranchParams {
	return sqlc.CreateBranchParams{
		ID:          b.ID().String(),
		WorkspaceID: b.WorkspaceID().String(),
		SeedID:      b.SeedID().String(),
		AuthorID:    b.AuthorID().String(),
		RootLeafID:  database.NullableString(b.RootLeafID().String()),
		CreatedAt:   b.Timestamps().CreatedAt.Time(),
	}
}

func branchToUpdateParams(b domain.Branch) sqlc.UpdateBranchParams {
	return sqlc.UpdateBranchParams{
		ID:         b.ID().String(),
		RootLeafID: database.NullableString(b.RootLeafID().String()),
	}
}

// --- branch: sqlc model → domain ---

func branchToDomain(row sqlc.Branch) domain.Branch {
	return domain.ReconstructBranch(
		types.BranchIDFrom(row.ID),
		types.WorkspaceIDFrom(row.WorkspaceID),
		types.SeedIDFrom(row.SeedID),
		types.UserIDFrom(row.AuthorID),
		types.LeafIDFrom(database.DerefString(row.RootLeafID)),
		types.Timestamps{CreatedAt: types.TimestampFrom(row.CreatedAt)},
	)
}

func branchesToDomain(rows []sqlc.Branch) []domain.Branch {
	branches := make([]domain.Branch, len(rows))
	for i, row := range rows {
		branches[i] = branchToDomain(row)
	}
	return branches
}

// --- connection: domain → sqlc params ---

func connectionToCreateParams(c domain.Connection) sqlc.CreateConnectionParams {
	return sqlc.CreateConnectionParams{
		ID:          c.ID().String(),
		WorkspaceID: c.WorkspaceID().String(),
		AuthorID:    c.AuthorID().String(),
		LeafIds:     leafIDsToStrings(c.LeafIDs()),
		CreatedAt:   c.Timestamps().CreatedAt.Time(),
	}
}

// --- connection: sqlc model → domain ---

func connectionToDomain(row sqlc.Connection) domain.Connection {
	return domain.ReconstructConnection(
		types.ConnectionIDFrom(row.ID),
		types.WorkspaceIDFrom(row.WorkspaceID),
		types.UserIDFrom(row.AuthorID),
		leafIDsFromStrings(row.LeafIds),
		types.Timestamps{CreatedAt: types.TimestampFrom(row.CreatedAt)},
	)
}

func connectionsToDomain(rows []sqlc.Connection) []domain.Connection {
	conns := make([]domain.Connection, len(rows))
	for i, row := range rows {
		conns[i] = connectionToDomain(row)
	}
	return conns
}

// --- leaf ID helpers ---

func leafIDsToStrings(ids []types.LeafID) []string {
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = id.String()
	}
	return strs
}

func leafIDsFromStrings(strs []string) []types.LeafID {
	ids := make([]types.LeafID, len(strs))
	for i, s := range strs {
		ids[i] = types.LeafIDFrom(s)
	}
	return ids
}
