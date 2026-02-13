package postgres

import (
	"github.com/0xsj/canopy-backend/internal/notification/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/notification/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// --- notification: domain → sqlc params ---

func notificationToCreateParams(n domain.Notification) sqlc.CreateNotificationParams {
	ts := n.Timestamps()
	return sqlc.CreateNotificationParams{
		ID:           n.ID().String(),
		UserID:       n.UserID().String(),
		Channel:      string(n.Channel()),
		Title:        n.Title(),
		Body:         database.NullableString(n.Body()),
		ResourceType: database.NullableString(n.ResourceType()),
		ResourceID:   database.NullableString(n.ResourceID()),
		WorkspaceID:  database.NullableString(n.WorkspaceID()),
		Status:       string(n.Status()),
		CreatedAt:    ts.CreatedAt.Time(),
		UpdatedAt:    database.TsUpdatedAt(ts),
	}
}

func notificationToUpdateParams(n domain.Notification) sqlc.UpdateNotificationParams {
	ts := n.Timestamps()
	return sqlc.UpdateNotificationParams{
		ID:        n.ID().String(),
		Status:    string(n.Status()),
		UpdatedAt: database.TsUpdatedAt(ts),
	}
}

// --- notification: sqlc model → domain ---

func notificationToDomain(row sqlc.Notification) domain.Notification {
	updatedTS := types.TimestampFrom(row.UpdatedAt)
	return domain.ReconstructNotification(
		domain.NotificationIDFrom(row.ID),
		types.UserIDFrom(row.UserID),
		domain.Channel(row.Channel),
		row.Title,
		database.DerefString(row.Body),
		database.DerefString(row.ResourceType),
		database.DerefString(row.ResourceID),
		database.DerefString(row.WorkspaceID),
		domain.NotificationStatus(row.Status),
		types.Timestamps{
			CreatedAt: types.TimestampFrom(row.CreatedAt),
			UpdatedAt: &updatedTS,
		},
	)
}

func notificationsToDomain(rows []sqlc.Notification) []domain.Notification {
	notifs := make([]domain.Notification, len(rows))
	for i, row := range rows {
		notifs[i] = notificationToDomain(row)
	}
	return notifs
}

// --- subscription: domain → sqlc params ---

func subscriptionToSaveParams(sub domain.Subscription) sqlc.SaveSubscriptionParams {
	ts := sub.Timestamps()
	channels := make([]string, len(sub.Channels()))
	for i, ch := range sub.Channels() {
		channels[i] = string(ch)
	}
	return sqlc.SaveSubscriptionParams{
		UserID:          sub.UserID().String(),
		WorkspaceID:     sub.WorkspaceID().String(),
		Channels:        channels,
		DigestFrequency: string(sub.DigestFrequency()),
		CreatedAt:       ts.CreatedAt.Time(),
		UpdatedAt:       database.TsUpdatedAt(ts),
	}
}

// --- subscription: sqlc model → domain ---

func subscriptionToDomain(row sqlc.Subscription) domain.Subscription {
	channels := make([]domain.Channel, len(row.Channels))
	for i, s := range row.Channels {
		channels[i] = domain.Channel(s)
	}
	updatedTS := types.TimestampFrom(row.UpdatedAt)
	return domain.ReconstructSubscription(
		types.UserIDFrom(row.UserID),
		types.WorkspaceIDFrom(row.WorkspaceID),
		channels,
		domain.DigestFrequency(row.DigestFrequency),
		types.Timestamps{
			CreatedAt: types.TimestampFrom(row.CreatedAt),
			UpdatedAt: &updatedTS,
		},
	)
}

func subscriptionsToDomain(rows []sqlc.Subscription) []domain.Subscription {
	subs := make([]domain.Subscription, len(rows))
	for i, row := range rows {
		subs[i] = subscriptionToDomain(row)
	}
	return subs
}
