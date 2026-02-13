package discussion

import (
	"github.com/0xsj/canopy-backend/internal/discussion/adapter/postgres"
	handler "github.com/0xsj/canopy-backend/internal/discussion/interface/http/v1"
	"github.com/0xsj/canopy-backend/internal/discussion/service"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// Provider encapsulates discussion context wiring.
type Provider struct {
	Service *service.Service
	Handler *handler.Handler
}

// Wire creates the discussion bounded context from infrastructure dependencies.
func Wire(db *database.DB, wsMembers service.WorkspaceMemberReader, pub events.Publisher, log logger.Logger) *Provider {
	dbtx := db.DBTX()
	repo := postgres.NewThreadRepository(dbtx)
	svc := service.New(repo, wsMembers, pub, log)
	h := handler.NewHandler(svc, log)
	return &Provider{Service: svc, Handler: h}
}
