package v1

import (
	"github.com/0xsj/canopy-backend/internal/identity/domain"
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
