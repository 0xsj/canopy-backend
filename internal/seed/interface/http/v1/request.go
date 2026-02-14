package v1

type PlantSeedRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type UpdateConstraintsRequest struct {
	Constraints map[string]any `json:"constraints"`
}

type UpdatePositionRequest struct {
	PositionX float64 `json:"position_x"`
	PositionY float64 `json:"position_y"`
}
