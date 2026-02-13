package postgres

import (
	"context"
	"time"

	"github.com/0xsj/canopy-backend/internal/notification/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// NotificationRepository implements domain.NotificationRepository using Postgres.
type NotificationRepository struct {
	db database.DBTX
}

// NewNotificationRepository creates a new NotificationRepository.
func NewNotificationRepository(db database.DBTX) *NotificationRepository {
	return &NotificationRepository{db: db}
}

var _ domain.NotificationRepository = (*NotificationRepository)(nil)

func (r *NotificationRepository) Create(ctx context.Context, notif domain.Notification) error {
	const op = "notification: create notification"
	const query = `
		INSERT INTO notifications (id, user_id, channel, title, body, resource_type, resource_id, workspace_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	ts := notif.Timestamps()
	_, err := r.db.Exec(ctx, query,
		notif.ID().String(),
		notif.UserID().String(),
		string(notif.Channel()),
		notif.Title(),
		nullableString(notif.Body()),
		nullableString(notif.ResourceType()),
		nullableString(notif.ResourceID()),
		nullableString(notif.WorkspaceID()),
		string(notif.Status()),
		ts.CreatedAt.Time(),
		tsUpdatedAt(ts),
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return nil
}

func (r *NotificationRepository) FindByUser(ctx context.Context, userID types.UserID, limit int) ([]domain.Notification, error) {
	const op = "notification: find notifications by user"
	const query = notificationSelectColumns + `
		FROM notifications WHERE user_id = $1
		ORDER BY created_at DESC LIMIT $2`

	return r.queryNotifications(ctx, query, op, userID.String(), limit)
}

func (r *NotificationRepository) FindUnreadByUser(ctx context.Context, userID types.UserID) ([]domain.Notification, error) {
	const op = "notification: find unread notifications by user"
	const query = notificationSelectColumns + `
		FROM notifications WHERE user_id = $1 AND status != 'read'
		ORDER BY created_at DESC`

	return r.queryNotifications(ctx, query, op, userID.String())
}

func (r *NotificationRepository) Update(ctx context.Context, notif domain.Notification) error {
	const op = "notification: update notification"
	const query = `UPDATE notifications SET status = $2, updated_at = $3 WHERE id = $1`

	ts := notif.Timestamps()
	tag, err := r.db.Exec(ctx, query,
		notif.ID().String(),
		string(notif.Status()),
		tsUpdatedAt(ts),
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}

func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID types.UserID) error {
	const op = "notification: mark all read"
	const query = `
		UPDATE notifications SET status = 'read', updated_at = NOW()
		WHERE user_id = $1 AND status != 'read'`

	_, err := r.db.Exec(ctx, query, userID.String())
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return nil
}

// --- helpers ---

type rowScanner interface {
	Scan(dest ...any) error
}

const notificationSelectColumns = `SELECT id, user_id, channel, title, body, resource_type, resource_id, workspace_id, status, created_at, updated_at`

func (r *NotificationRepository) queryNotifications(ctx context.Context, query, op string, args ...any) ([]domain.Notification, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	defer rows.Close()

	var notifications []domain.Notification
	for rows.Next() {
		n, err := scanNotification(rows, op)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	return notifications, rows.Err()
}

func scanNotification(row rowScanner, op string) (domain.Notification, error) {
	var (
		rawID        string
		rawUserID    string
		channel      string
		title        string
		body         *string
		resourceType *string
		resourceID   *string
		workspaceID  *string
		status       string
		createdAt    time.Time
		updatedAt    time.Time
	)

	if err := row.Scan(&rawID, &rawUserID, &channel, &title, &body, &resourceType, &resourceID, &workspaceID, &status, &createdAt, &updatedAt); err != nil {
		return domain.Notification{}, database.MapQueryError(err, op)
	}

	updatedTS := types.TimestampFrom(updatedAt)
	return domain.ReconstructNotification(
		domain.NotificationIDFrom(rawID),
		types.UserIDFrom(rawUserID),
		domain.Channel(channel),
		title,
		derefString(body),
		derefString(resourceType),
		derefString(resourceID),
		derefString(workspaceID),
		domain.NotificationStatus(status),
		types.Timestamps{
			CreatedAt: types.TimestampFrom(createdAt),
			UpdatedAt: &updatedTS,
		},
	), nil
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func tsUpdatedAt(ts types.Timestamps) time.Time {
	if ts.UpdatedAt != nil {
		return ts.UpdatedAt.Time()
	}
	return ts.CreatedAt.Time()
}
