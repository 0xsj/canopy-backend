package v1

type CreateLeafRequest struct {
	SeedID        string   `json:"seed_id"`
	BranchID      string   `json:"branch_id"`
	ParentLeafID  string   `json:"parent_leaf_id"`
	Title         string   `json:"title"`
	Summary       string   `json:"summary"`
	KeyPoints     []string `json:"key_points,omitempty"`
	OpenQuestions []string `json:"open_questions,omitempty"`
	Tags          []string `json:"tags,omitempty"`
}

type StartBranchRequest struct {
	SeedID        string   `json:"seed_id"`
	Title         string   `json:"title"`
	Summary       string   `json:"summary"`
	KeyPoints     []string `json:"key_points,omitempty"`
	OpenQuestions []string `json:"open_questions,omitempty"`
	Tags          []string `json:"tags,omitempty"`
}

type CreateConnectionRequest struct {
	LeafIDs []string `json:"leaf_ids"`
}
