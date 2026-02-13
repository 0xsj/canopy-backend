package identity

import (
	"github.com/0xsj/canopy-backend/internal/identity/adapter/postgres"
	handler "github.com/0xsj/canopy-backend/internal/identity/interface/http/v1"
	"github.com/0xsj/canopy-backend/internal/identity/service"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// Provider encapsulates identity context wiring.
type Provider struct {
	Service  *service.Service
	Handler  *handler.Handler
	UserRepo *postgres.UserRepository // exposed: org needs this for UserReader adapter
}

// Wire creates the identity bounded context from infrastructure dependencies.
func Wire(db *database.DB, pub events.Publisher, log logger.Logger) *Provider {
	dbtx := db.DBTX()
	repo := postgres.NewUserRepository(dbtx)
	svc := service.New(repo, pub, log)
	h := handler.NewHandler(svc, log)
	return &Provider{Service: svc, Handler: h, UserRepo: repo}
}
