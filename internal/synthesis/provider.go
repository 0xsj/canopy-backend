package synthesis

import (
	"github.com/0xsj/canopy-backend/internal/synthesis/adapter/postgres"
	handler "github.com/0xsj/canopy-backend/internal/synthesis/interface/http/v1"
	"github.com/0xsj/canopy-backend/internal/synthesis/service"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/llm"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// Provider encapsulates synthesis context wiring.
type Provider struct {
	Service *service.Service
	Handler *handler.Handler
}

// Wire creates the synthesis bounded context from infrastructure dependencies.
func Wire(
	db *database.DB,
	llmResolver llm.ProviderResolver,
	broadcaster llm.StreamBroadcaster,
	leaves service.SourceLeafReader,
	seeds service.SeedReader,
	leafCreator service.SynthesisLeafCreator,
	wsMembers service.WorkspaceMemberReader,
	pub events.Publisher,
	log logger.Logger,
) *Provider {
	dbtx := db.DBTX()
	repo := postgres.NewSynthesisRepository(dbtx)
	svc := service.New(repo, llmResolver, broadcaster, leaves, seeds, leafCreator, wsMembers, pub, log)
	h := handler.NewHandler(svc, log)
	return &Provider{Service: svc, Handler: h}
}
