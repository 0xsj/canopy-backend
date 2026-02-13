package exploration

import (
	"github.com/0xsj/canopy-backend/internal/exploration/adapter/postgres"
	"github.com/0xsj/canopy-backend/internal/exploration/domain"
	handler "github.com/0xsj/canopy-backend/internal/exploration/interface/http/v1"
	"github.com/0xsj/canopy-backend/internal/exploration/service"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// Provider encapsulates exploration context wiring.
type Provider struct {
	Service  *service.Service
	Handler  *handler.Handler
	LeafRepo *postgres.LeafRepository // exposed: convergence needs this for LeafWriter adapter
}

// Wire creates the exploration bounded context from infrastructure dependencies.
func Wire(db *database.DB, wsMembers service.WorkspaceMemberReader, pub events.Publisher, log logger.Logger) *Provider {
	dbtx := db.DBTX()
	leafRepo := postgres.NewLeafRepository(dbtx)
	branchRepo := postgres.NewBranchRepository(dbtx)
	connRepo := postgres.NewConnectionRepository(dbtx)

	svc := service.New(
		leafRepo, branchRepo, connRepo,
		wsMembers,
		db,
		func(tx database.DBTX) domain.LeafRepository { return postgres.NewLeafRepository(tx) },
		func(tx database.DBTX) domain.BranchRepository { return postgres.NewBranchRepository(tx) },
		pub, log,
	)
	h := handler.NewHandler(svc, log)
	return &Provider{Service: svc, Handler: h, LeafRepo: leafRepo}
}
