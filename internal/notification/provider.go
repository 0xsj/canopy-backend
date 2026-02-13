package notification

import (
	"github.com/0xsj/canopy-backend/internal/notification/adapter/postgres"
	"github.com/0xsj/canopy-backend/internal/notification/domain"
	handler "github.com/0xsj/canopy-backend/internal/notification/interface/http/v1"
	"github.com/0xsj/canopy-backend/internal/notification/service"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// Provider encapsulates notification context wiring.
type Provider struct {
	Service *service.Service
	Handler *handler.Handler
}

// Wire creates the notification bounded context from infrastructure dependencies.
func Wire(db *database.DB, transports []domain.NotificationTransport, pub events.Publisher, log logger.Logger) *Provider {
	dbtx := db.DBTX()
	notifRepo := postgres.NewNotificationRepository(dbtx)
	subRepo := postgres.NewSubscriptionRepository(dbtx)
	svc := service.New(notifRepo, subRepo, transports, pub, log)
	h := handler.NewHandler(svc, log)
	return &Provider{Service: svc, Handler: h}
}
