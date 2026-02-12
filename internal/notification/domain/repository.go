package domain

import (
	"context"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// NotificationRepository defines the persistence port for notifications.
type NotificationRepository interface {
	// Create persists a new notification.
	Create(ctx context.Context, notification Notification) error

	// FindByUser returns notifications for a user, ordered by most recent.
	FindByUser(ctx context.Context, userID types.UserID, limit int) ([]Notification, error)

	// FindUnreadByUser returns unread notifications for a user.
	FindUnreadByUser(ctx context.Context, userID types.UserID) ([]Notification, error)

	// Update persists status changes to a notification.
	Update(ctx context.Context, notification Notification) error

	// MarkAllRead marks all pending/delivered notifications as read for a user.
	MarkAllRead(ctx context.Context, userID types.UserID) error
}

// SubscriptionRepository defines the persistence port for notification subscriptions.
type SubscriptionRepository interface {
	// Save persists a subscription (upsert — create or update).
	Save(ctx context.Context, subscription Subscription) error

	// FindByUserAndWorkspace returns a user's subscription for a workspace.
	FindByUserAndWorkspace(ctx context.Context, userID types.UserID, workspaceID types.WorkspaceID) (Subscription, error)

	// FindByWorkspace returns all subscriptions for a workspace.
	// Used when determining who to notify about a workspace event.
	FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]Subscription, error)
}

// NotificationTransport defines the port for delivering notifications
// through a specific channel (in-app, push, email).
type NotificationTransport interface {
	// Send delivers a notification through this transport's channel.
	Send(ctx context.Context, notification Notification) error

	// Channel returns which channel this transport handles.
	Channel() Channel
}
