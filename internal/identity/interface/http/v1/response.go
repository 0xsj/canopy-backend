package v1

import (
	"github.com/0xsj/canopy-backend/internal/identity/domain"
	"github.com/0xsj/canopy-backend/internal/identity/service"
	"github.com/0xsj/canopy-backend/pkg/types"
)

type UserResponse struct {
	ID          string           `json:"id"`
	ExternalID  string           `json:"external_id"`
	DisplayName string           `json:"display_name"`
	Email       string           `json:"email"`
	AvatarURL   string           `json:"avatar_url"`
	CreatedAt   types.Timestamp  `json:"created_at"`
	UpdatedAt   *types.Timestamp `json:"updated_at,omitempty"`
}

func UserFromDomain(u domain.User) UserResponse {
	ts := u.Timestamps()
	return UserResponse{
		ID:          u.ID().String(),
		ExternalID:  u.ExternalID(),
		DisplayName: u.DisplayName(),
		Email:       u.Email(),
		AvatarURL:   u.AvatarURL(),
		CreatedAt:   ts.CreatedAt,
		UpdatedAt:   ts.UpdatedAt,
	}
}

type UserLLMConfigResponse struct {
	Provider  string           `json:"provider"`
	Model     string           `json:"model"`
	APIKey    string           `json:"api_key"`
	CreatedAt types.Timestamp  `json:"created_at"`
	UpdatedAt *types.Timestamp `json:"updated_at,omitempty"`
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:3] + "..." + key[len(key)-4:]
}

func UserLLMConfigFromResult(r service.UserLLMConfigResult) UserLLMConfigResponse {
	return UserLLMConfigResponse{
		Provider:  r.Provider,
		Model:     r.Model,
		APIKey:    maskAPIKey(r.APIKey),
		CreatedAt: r.Timestamps.CreatedAt,
		UpdatedAt: r.Timestamps.UpdatedAt,
	}
}
