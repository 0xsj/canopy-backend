package session

import (
	"github.com/0xsj/canopy-backend/internal/session/adapter/postgres"
	"github.com/0xsj/canopy-backend/internal/session/domain"
	handler "github.com/0xsj/canopy-backend/internal/session/interface/http/v1"
	"github.com/0xsj/canopy-backend/internal/session/service"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// Provider encapsulates session context wiring.
type Provider struct {
	Service *service.Service
	Handler *handler.Handler
}

// Wire creates the session bounded context from infrastructure dependencies.
func Wire(db *database.DB, assembler domain.ContextAssembler, wsMembers service.WorkspaceMemberReader, pub events.Publisher, log logger.Logger) *Provider {
	dbtx := db.DBTX()
	repo := postgres.NewSessionRepository(dbtx)
	svc := service.New(repo, assembler, wsMembers, pub, log)
	h := handler.NewHandler(svc, log)
	return &Provider{Service: svc, Handler: h}
}
