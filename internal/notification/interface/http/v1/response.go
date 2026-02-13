package v1

import (
	"github.com/0xsj/canopy-backend/internal/notification/domain"
	"github.com/0xsj/canopy-backend/pkg/types"
)

type NotificationResponse struct {
	ID           string           `json:"id"`
	UserID       string           `json:"user_id"`
	Channel      string           `json:"channel"`
	Title        string           `json:"title"`
	Body         string           `json:"body"`
	ResourceType string           `json:"resource_type"`
	ResourceID   string           `json:"resource_id"`
	WorkspaceID  string           `json:"workspace_id,omitempty"`
	Status       string           `json:"status"`
	CreatedAt    types.Timestamp  `json:"created_at"`
	UpdatedAt    *types.Timestamp `json:"updated_at,omitempty"`
}

func NotificationFromDomain(n domain.Notification) NotificationResponse {
	ts := n.Timestamps()
	return NotificationResponse{
		ID:           n.ID().String(),
		UserID:       n.UserID().String(),
		Channel:      string(n.Channel()),
		Title:        n.Title(),
		Body:         n.Body(),
		ResourceType: n.ResourceType(),
		ResourceID:   n.ResourceID(),
		WorkspaceID:  n.WorkspaceID(),
		Status:       string(n.Status()),
		CreatedAt:    ts.CreatedAt,
		UpdatedAt:    ts.UpdatedAt,
	}
}

func NotificationsFromDomain(notifications []domain.Notification) []NotificationResponse {
	out := make([]NotificationResponse, len(notifications))
	for i, n := range notifications {
		out[i] = NotificationFromDomain(n)
	}
	return out
}
