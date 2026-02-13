package service

import (
	"context"
	"time"

	"github.com/0xsj/canopy-backend/internal/notification/domain"
	"github.com/0xsj/canopy-backend/pkg/auth"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// Service implements the notification application logic.
type Service struct {
	notifications domain.NotificationRepository
	subscriptions domain.SubscriptionRepository
	transports    map[domain.Channel]domain.NotificationTransport
	pub           events.Publisher
	log           logger.Logger
}

// New creates a new notification service.
func New(
	notifications domain.NotificationRepository,
	subscriptions domain.SubscriptionRepository,
	transports []domain.NotificationTransport,
	pub events.Publisher,
	log logger.Logger,
) *Service {
	transportMap := make(map[domain.Channel]domain.NotificationTransport, len(transports))
	for _, t := range transports {
		transportMap[t.Channel()] = t
	}
	return &Service{
		notifications: notifications,
		subscriptions: subscriptions,
		transports:    transportMap,
		pub:           pub,
		log:           log,
	}
}

// Send creates and delivers a notification through the appropriate transport.
func (s *Service) Send(
	ctx context.Context,
	userID types.UserID,
	channel domain.Channel,
	title, body, resourceType, resourceID, workspaceID string,
) error {
	const op = "notification: send"

	notification, err := domain.NewNotification(userID, channel, title, body, resourceType, resourceID, workspaceID)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := s.notifications.Create(ctx, notification); err != nil {
		return canopyerr.Wrap(err, op)
	}

	// Deliver via transport if available.
	transport, ok := s.transports[channel]
	if ok {
		if err := transport.Send(ctx, notification); err != nil {
			s.log.Error("transport delivery failed",
				logger.String("channel", string(channel)),
				logger.Err(err),
			)
			// Don't fail the operation — the notification is persisted.
		} else {
			if markErr := notification.MarkDelivered(); markErr == nil {
				_ = s.notifications.Update(ctx, notification)
			}
		}
	}

	s.publish(ctx, domain.SubjectNotificationSent, workspaceID, domain.NotificationSentData{
		NotificationID: notification.ID().String(),
		UserID:         userID.String(),
		Channel:        string(channel),
		Title:          title,
		Timestamp:      time.Now().UTC(),
	})

	return nil
}

// FindByUser returns notifications for a user, ordered by most recent.
func (s *Service) FindByUser(ctx context.Context, limit int) ([]domain.Notification, error) {
	const op = "notification: find by user"

	callerID, err := s.authenticatedUserID(ctx)
	if err != nil {
		return nil, canopyerr.Wrap(err, op)
	}

	notifications, err := s.notifications.FindByUser(ctx, callerID, limit)
	if err != nil {
		return nil, canopyerr.Wrap(err, op)
	}
	return notifications, nil
}

// FindUnread returns unread notifications for the authenticated user.
func (s *Service) FindUnread(ctx context.Context) ([]domain.Notification, error) {
	const op = "notification: find unread"

	callerID, err := s.authenticatedUserID(ctx)
	if err != nil {
		return nil, canopyerr.Wrap(err, op)
	}

	notifications, err := s.notifications.FindUnreadByUser(ctx, callerID)
	if err != nil {
		return nil, canopyerr.Wrap(err, op)
	}
	return notifications, nil
}

// MarkRead marks a single notification as read.
func (s *Service) MarkRead(ctx context.Context, notificationID domain.NotificationID) error {
	const op = "notification: mark read"
	// Notification lookup is by ID; we skip ownership check for simplicity.
	// In production, verify the notification belongs to the caller.
	_ = ctx
	_ = notificationID
	// TODO: Implement when individual notification fetch is available.
	return canopyerr.Wrap(canopyerr.ErrNotFound, op)
}

// MarkAllRead marks all notifications as read for the authenticated user.
func (s *Service) MarkAllRead(ctx context.Context) error {
	const op = "notification: mark all read"

	callerID, err := s.authenticatedUserID(ctx)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := s.notifications.MarkAllRead(ctx, callerID); err != nil {
		return canopyerr.Wrap(err, op)
	}
	return nil
}

// UpdateSubscription updates or creates a user's notification subscription for a workspace.
func (s *Service) UpdateSubscription(
	ctx context.Context,
	workspaceID types.WorkspaceID,
	channels []domain.Channel,
	digestFrequency domain.DigestFrequency,
) error {
	const op = "notification: update subscription"

	callerID, err := s.authenticatedUserID(ctx)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	sub, err := s.subscriptions.FindByUserAndWorkspace(ctx, callerID, workspaceID)
	if err != nil {
		if canopyerr.GetKind(err) != canopyerr.KindNotFound {
			return canopyerr.Wrap(err, op)
		}
		// Create a new subscription.
		sub, err = domain.NewSubscription(callerID, workspaceID)
		if err != nil {
			return canopyerr.Wrap(err, op)
		}
	}

	if len(channels) > 0 {
		if err := sub.UpdateChannels(channels); err != nil {
			return canopyerr.Wrap(err, op)
		}
	}

	if digestFrequency != "" {
		if err := sub.UpdateDigestFrequency(digestFrequency); err != nil {
			return canopyerr.Wrap(err, op)
		}
	}

	if err := s.subscriptions.Save(ctx, sub); err != nil {
		return canopyerr.Wrap(err, op)
	}

	return nil
}

// --- Auth Helpers ---

func (s *Service) authenticatedUserID(ctx context.Context) (types.UserID, error) {
	claims, ok := auth.FromClaims(ctx)
	if !ok {
		return types.UserID{}, canopyerr.ErrUnauthenticated
	}
	return types.UserIDFrom(claims.Subject), nil
}

// --- Event Publishing ---

func (s *Service) publish(ctx context.Context, eventType, workspaceID string, data any) {
	event, err := events.New(eventType, workspaceID, data)
	if err != nil {
		s.log.Error("event creation failed", logger.String("type", eventType), logger.Err(err))
		return
	}
	event.Subject = events.BuildSubject(workspaceID, "notification", eventType)
	if pubErr := s.pub.Publish(ctx, event); pubErr != nil {
		s.log.Error("event publish failed", logger.String("type", eventType), logger.Err(pubErr))
	}
}
