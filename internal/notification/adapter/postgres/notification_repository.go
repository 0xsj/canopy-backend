package postgres

import (
	"context"

	"github.com/0xsj/canopy-backend/internal/notification/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/notification/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// NotificationRepository implements domain.NotificationRepository using Postgres via sqlc.
type NotificationRepository struct {
	q *sqlc.Queries
}

// NewNotificationRepository creates a new NotificationRepository.
func NewNotificationRepository(db database.DBTX) *NotificationRepository {
	return &NotificationRepository{q: sqlc.New(db)}
}

var _ domain.NotificationRepository = (*NotificationRepository)(nil)

func (r *NotificationRepository) Create(ctx context.Context, notif domain.Notification) error {
	const op = "notification: create notification"
	return database.MapQueryError(r.q.CreateNotification(ctx, notificationToCreateParams(notif)), op)
}

func (r *NotificationRepository) FindByID(ctx context.Context, id domain.NotificationID) (domain.Notification, error) {
	const op = "notification: find notification by id"
	row, err := r.q.FindNotificationByID(ctx, id.String())
	if err != nil {
		return domain.Notification{}, database.MapQueryError(err, op)
	}
	return notificationToDomain(row), nil
}

func (r *NotificationRepository) FindByUser(ctx context.Context, userID types.UserID, limit int) ([]domain.Notification, error) {
	const op = "notification: find notifications by user"
	rows, err := r.q.FindNotificationsByUser(ctx, sqlc.FindNotificationsByUserParams{
		UserID: userID.String(),
		Limit:  limit,
	})
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return notificationsToDomain(rows), nil
}

func (r *NotificationRepository) FindUnreadByUser(ctx context.Context, userID types.UserID) ([]domain.Notification, error) {
	const op = "notification: find unread notifications by user"
	rows, err := r.q.FindUnreadNotificationsByUser(ctx, userID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return notificationsToDomain(rows), nil
}

func (r *NotificationRepository) Update(ctx context.Context, notif domain.Notification) error {
	const op = "notification: update notification"
	tag, err := r.q.UpdateNotification(ctx, notificationToUpdateParams(notif))
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
	return database.MapQueryError(r.q.MarkAllNotificationsRead(ctx, userID.String()), op)
}
