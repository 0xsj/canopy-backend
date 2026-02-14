package v1

import (
	"github.com/0xsj/canopy-backend/internal/exploration/domain"
	"github.com/0xsj/canopy-backend/pkg/types"
)

type LeafResponse struct {
	ID            string          `json:"id"`
	WorkspaceID   string          `json:"workspace_id"`
	SeedID        string          `json:"seed_id"`
	BranchID      string          `json:"branch_id"`
	AuthorID      string          `json:"author_id"`
	ParentLeafID  string          `json:"parent_leaf_id,omitempty"`
	Title         string          `json:"title"`
	Summary       string          `json:"summary"`
	KeyPoints     []string        `json:"key_points,omitempty"`
	OpenQuestions []string        `json:"open_questions,omitempty"`
	Tags          []string        `json:"tags,omitempty"`
	Layer         string          `json:"layer"`
	Sources       []domain.Source `json:"sources,omitempty"`
	Metadata      map[string]any  `json:"metadata,omitempty"`
	PositionX     float64         `json:"position_x"`
	PositionY     float64         `json:"position_y"`
	CreatedAt     types.Timestamp `json:"created_at"`
}

func LeafFromDomain(l domain.Leaf) LeafResponse {
	ts := l.Timestamps()
	return LeafResponse{
		ID:            l.ID().String(),
		WorkspaceID:   l.WorkspaceID().String(),
		SeedID:        l.SeedID().String(),
		BranchID:      l.BranchID().String(),
		AuthorID:      l.AuthorID().String(),
		ParentLeafID:  l.ParentLeafID().String(),
		Title:         l.Title(),
		Summary:       l.Summary(),
		KeyPoints:     l.KeyPoints(),
		OpenQuestions: l.OpenQuestions(),
		Tags:          l.Tags(),
		Layer:         string(l.Layer()),
		Sources:       l.Sources(),
		Metadata:      l.Metadata(),
		PositionX:     l.PositionX(),
		PositionY:     l.PositionY(),
		CreatedAt:     ts.CreatedAt,
	}
}

func LeavesFromDomain(leaves []domain.Leaf) []LeafResponse {
	out := make([]LeafResponse, len(leaves))
	for i, l := range leaves {
		out[i] = LeafFromDomain(l)
	}
	return out
}

type BranchResponse struct {
	ID          string          `json:"id"`
	WorkspaceID string          `json:"workspace_id"`
	SeedID      string          `json:"seed_id"`
	AuthorID    string          `json:"author_id"`
	RootLeafID  string          `json:"root_leaf_id"`
	CreatedAt   types.Timestamp `json:"created_at"`
}

func BranchFromDomain(b domain.Branch) BranchResponse {
	ts := b.Timestamps()
	return BranchResponse{
		ID:          b.ID().String(),
		WorkspaceID: b.WorkspaceID().String(),
		SeedID:      b.SeedID().String(),
		AuthorID:    b.AuthorID().String(),
		RootLeafID:  b.RootLeafID().String(),
		CreatedAt:   ts.CreatedAt,
	}
}

type ConnectionResponse struct {
	ID          string          `json:"id"`
	WorkspaceID string          `json:"workspace_id"`
	AuthorID    string          `json:"author_id"`
	LeafIDs     []string        `json:"leaf_ids"`
	CreatedAt   types.Timestamp `json:"created_at"`
}

func ConnectionFromDomain(c domain.Connection) ConnectionResponse {
	leafIDs := make([]string, len(c.LeafIDs()))
	for i, id := range c.LeafIDs() {
		leafIDs[i] = id.String()
	}
	ts := c.Timestamps()
	return ConnectionResponse{
		ID:          c.ID().String(),
		WorkspaceID: c.WorkspaceID().String(),
		AuthorID:    c.AuthorID().String(),
		LeafIDs:     leafIDs,
		CreatedAt:   ts.CreatedAt,
	}
}

func ConnectionsFromDomain(conns []domain.Connection) []ConnectionResponse {
	out := make([]ConnectionResponse, len(conns))
	for i, c := range conns {
		out[i] = ConnectionFromDomain(c)
	}
	return out
}

type StartBranchResponse struct {
	Branch BranchResponse `json:"branch"`
	Leaf   LeafResponse   `json:"leaf"`
}
