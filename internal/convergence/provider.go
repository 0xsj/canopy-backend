package convergence

import (
	"github.com/0xsj/canopy-backend/internal/convergence/adapter/postgres"
	handler "github.com/0xsj/canopy-backend/internal/convergence/interface/http/v1"
	"github.com/0xsj/canopy-backend/internal/convergence/service"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// Provider encapsulates convergence context wiring.
type Provider struct {
	Service *service.Service
	Handler *handler.Handler
}

// Wire creates the convergence bounded context from infrastructure dependencies.
func Wire(db *database.DB, leafWriter service.LeafWriter, wsMembers service.WorkspaceMemberReader, pub events.Publisher, log logger.Logger) *Provider {
	dbtx := db.DBTX()
	signalRepo := postgres.NewSignalRepository(dbtx)
	checkpointRepo := postgres.NewCheckpointRepository(dbtx)
	svc := service.New(signalRepo, checkpointRepo, leafWriter, wsMembers, pub, log)
	h := handler.NewHandler(svc, log)
	return &Provider{Service: svc, Handler: h}
}
