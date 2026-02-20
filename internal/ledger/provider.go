package ledger

import (
	"github.com/0xsj/canopy-backend/internal/ledger/adapter/postgres"
	handler "github.com/0xsj/canopy-backend/internal/ledger/interface/http/v1"
	"github.com/0xsj/canopy-backend/internal/ledger/service"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// Provider encapsulates ledger context wiring.
type Provider struct {
	Service *service.Service
	Handler *handler.Handler
}

// Wire creates the ledger bounded context from infrastructure dependencies.
// Ledger is a sink — no event publisher needed.
func Wire(db *database.DB, wsMembers service.WorkspaceMemberReader, log logger.Logger) *Provider {
	dbtx := db.DBTX()
	repo := postgres.NewLedgerRepository(dbtx)
	svc := service.New(repo, wsMembers, log)
	h := handler.NewHandler(svc, log)
	return &Provider{Service: svc, Handler: h}
}
