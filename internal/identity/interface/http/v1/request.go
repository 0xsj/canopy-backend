package v1

type RegisterRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

type UpdateProfileRequest struct {
	DisplayName *string `json:"display_name,omitempty"`
	Email       *string `json:"email,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

type SetUserLLMConfigRequest struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	APIKey   string `json:"api_key"`
}
