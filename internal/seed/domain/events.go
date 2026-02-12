package domain

import "time"

// Event subjects published by the seed context.
const (
	SubjectSeedPlanted            = "seed.planted"
	SubjectSeedConstraintsUpdated = "seed.constraints.updated"
)

// SeedPlantedData is published when a new seed is planted in a workspace.
type SeedPlantedData struct {
	SeedID      string    `json:"seed_id"`
	WorkspaceID string    `json:"workspace_id"`
	AuthorID    string    `json:"author_id"`
	Title       string    `json:"title"`
	Timestamp   time.Time `json:"timestamp"`
}

// SeedConstraintsUpdatedData is published when a seed's constraints change.
type SeedConstraintsUpdatedData struct {
	SeedID      string    `json:"seed_id"`
	WorkspaceID string    `json:"workspace_id"`
	Timestamp   time.Time `json:"timestamp"`
}
