package inapp

import (
	"context"

	"github.com/0xsj/canopy-backend/internal/notification/domain"
)

// Transport implements NotificationTransport for the in-app channel.
// For in-app notifications, the notification is already persisted to the
// database by the service layer. "Delivery" means it's available for
// the client to fetch via the API. No external push is needed.
type Transport struct{}

// NewTransport creates a new in-app notification transport.
func NewTransport() *Transport {
	return &Transport{}
}

var _ domain.NotificationTransport = (*Transport)(nil)

func (t *Transport) Send(_ context.Context, _ domain.Notification) error {
	// No-op: the notification is already in the database and queryable
	// by the client via GET /api/v1/notifications endpoints.
	return nil
}

func (t *Transport) Channel() domain.Channel {
	return domain.ChannelInApp
}
