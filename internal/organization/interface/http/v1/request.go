package v1

type CreateOrgRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type AddMemberRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

type UpdateMemberRoleRequest struct {
	Role string `json:"role"`
}

type CreateTeamRequest struct {
	Name string `json:"name"`
}

type AddTeamMemberRequest struct {
	UserID string `json:"user_id"`
}
