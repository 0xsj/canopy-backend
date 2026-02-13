package seed

import (
	"github.com/0xsj/canopy-backend/internal/seed/adapter/postgres"
	handler "github.com/0xsj/canopy-backend/internal/seed/interface/http/v1"
	"github.com/0xsj/canopy-backend/internal/seed/service"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// Provider encapsulates seed context wiring.
type Provider struct {
	Service *service.Service
	Handler *handler.Handler
}

// Wire creates the seed bounded context from infrastructure dependencies.
func Wire(db *database.DB, wsMembers service.WorkspaceMemberReader, pub events.Publisher, log logger.Logger) *Provider {
	dbtx := db.DBTX()
	repo := postgres.NewSeedRepository(dbtx)
	svc := service.New(repo, wsMembers, pub, log)
	h := handler.NewHandler(svc, log)
	return &Provider{Service: svc, Handler: h}
}
