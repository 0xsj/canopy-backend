package v1

type PlantSeedRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type UpdateConstraintsRequest struct {
	Constraints map[string]any `json:"constraints"`
}
