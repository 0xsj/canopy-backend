package v1

type CreateWorkspaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateRoleRequest struct {
	Role string `json:"role"`
}

type UpdateConfigRequest struct {
	LoreKeeperMode string `json:"lore_keeper_mode"`
	LoreKeeperID   string `json:"lore_keeper_id"`
}

type TransitionPhaseRequest struct {
	Target string `json:"target"`
}
