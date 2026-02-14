package v1

import (
	"github.com/0xsj/canopy-backend/internal/seed/domain"
	"github.com/0xsj/canopy-backend/pkg/types"
)

type SeedResponse struct {
	ID          string           `json:"id"`
	WorkspaceID string           `json:"workspace_id"`
	AuthorID    string           `json:"author_id"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Constraints map[string]any   `json:"constraints,omitempty"`
	Tags        []string         `json:"tags,omitempty"`
	PositionX   float64          `json:"position_x"`
	PositionY   float64          `json:"position_y"`
	CreatedAt   types.Timestamp  `json:"created_at"`
	UpdatedAt   *types.Timestamp `json:"updated_at,omitempty"`
}

func SeedFromDomain(s domain.Seed) SeedResponse {
	ts := s.Timestamps()
	return SeedResponse{
		ID:          s.ID().String(),
		WorkspaceID: s.WorkspaceID().String(),
		AuthorID:    s.AuthorID().String(),
		Title:       s.Title(),
		Description: s.Description(),
		Constraints: s.Constraints(),
		Tags:        s.Tags(),
		PositionX:   s.PositionX(),
		PositionY:   s.PositionY(),
		CreatedAt:   ts.CreatedAt,
		UpdatedAt:   ts.UpdatedAt,
	}
}

func SeedsFromDomain(seeds []domain.Seed) []SeedResponse {
	out := make([]SeedResponse, len(seeds))
	for i, s := range seeds {
		out[i] = SeedFromDomain(s)
	}
	return out
}
