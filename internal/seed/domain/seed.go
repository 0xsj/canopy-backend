package domain

import (
	"fmt"
	"strings"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// Seed defines the starting topic, constraints, and boundary rules for
// exploration within a workspace. Seeds are planted by the lore keeper
// and scoped to a workspace. The title and description are immutable
// after creation; constraints and tags can be updated by the lore keeper.
type Seed struct {
	id          types.SeedID
	workspaceID types.WorkspaceID
	authorID    types.UserID
	title       string
	description string
	constraints map[string]any
	tags        []string
	positionX   float64
	positionY   float64
	timestamps  types.Timestamps
}

// NewSeed creates a new seed within a workspace.
func NewSeed(workspaceID types.WorkspaceID, authorID types.UserID, title, description string) (Seed, error) {
	if workspaceID.IsZero() {
		return Seed{}, fmt.Errorf("seed: workspace ID is required")
	}
	if authorID.IsZero() {
		return Seed{}, fmt.Errorf("seed: author ID is required")
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return Seed{}, fmt.Errorf("seed: title is required")
	}

	return Seed{
		id:          types.NewSeedID(),
		workspaceID: workspaceID,
		authorID:    authorID,
		title:       title,
		description: strings.TrimSpace(description),
		constraints: make(map[string]any),
		tags:        nil,
		timestamps:  types.NewMutableTimestamps(),
	}, nil
}

// ReconstructSeed builds a Seed from trusted data.
func ReconstructSeed(
	id types.SeedID,
	workspaceID types.WorkspaceID,
	authorID types.UserID,
	title string,
	description string,
	constraints map[string]any,
	tags []string,
	positionX, positionY float64,
	timestamps types.Timestamps,
) Seed {
	if constraints == nil {
		constraints = make(map[string]any)
	}
	return Seed{
		id:          id,
		workspaceID: workspaceID,
		authorID:    authorID,
		title:       title,
		description: description,
		constraints: constraints,
		tags:        tags,
		positionX:   positionX,
		positionY:   positionY,
		timestamps:  timestamps,
	}
}

// UpdatePosition sets the seed's canvas position.
func (s *Seed) UpdatePosition(x, y float64) {
	s.positionX = x
	s.positionY = y
	s.timestamps.Touch()
}

// UpdateConstraints replaces the seed's constraints. Only the lore keeper
// should call this — authorization is enforced at the service layer.
func (s *Seed) UpdateConstraints(constraints map[string]any) {
	if constraints == nil {
		constraints = make(map[string]any)
	}
	s.constraints = constraints
	s.timestamps.Touch()
}

// UpdateTags replaces the seed's tags.
func (s *Seed) UpdateTags(tags []string) {
	s.tags = tags
	s.timestamps.Touch()
}

func (s Seed) ID() types.SeedID               { return s.id }
func (s Seed) WorkspaceID() types.WorkspaceID { return s.workspaceID }
func (s Seed) AuthorID() types.UserID         { return s.authorID }
func (s Seed) Title() string                  { return s.title }
func (s Seed) Description() string            { return s.description }
func (s Seed) Constraints() map[string]any    { return s.constraints }
func (s Seed) Tags() []string                 { return s.tags }
func (s Seed) PositionX() float64             { return s.positionX }
func (s Seed) PositionY() float64             { return s.positionY }
func (s Seed) Timestamps() types.Timestamps   { return s.timestamps }
