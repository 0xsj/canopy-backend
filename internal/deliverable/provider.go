package deliverable

import (
	"github.com/0xsj/canopy-backend/internal/deliverable/adapter/postgres"
	handler "github.com/0xsj/canopy-backend/internal/deliverable/interface/http/v1"
	"github.com/0xsj/canopy-backend/internal/deliverable/service"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/llm"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// Provider encapsulates deliverable context wiring.
type Provider struct {
	Service *service.Service
	Handler *handler.Handler
}

// Wire creates the deliverable bounded context from infrastructure dependencies.
func Wire(
	db *database.DB,
	resolver llm.ProviderResolver,
	leaves service.SourceLeafReader,
	wsConfig service.WorkspaceConfigReader,
	wsMembers service.WorkspaceMemberReader,
	pub events.Publisher,
	log logger.Logger,
) *Provider {
	dbtx := db.DBTX()
	repo := postgres.NewDeliverableRepository(dbtx)
	svc := service.New(repo, resolver, leaves, wsConfig, wsMembers, pub, log)
	h := handler.NewHandler(svc, log)
	return &Provider{Service: svc, Handler: h}
}
