package organization

import (
	"github.com/0xsj/canopy-backend/internal/organization/adapter/postgres"
	"github.com/0xsj/canopy-backend/internal/organization/domain"
	handler "github.com/0xsj/canopy-backend/internal/organization/interface/http/v1"
	"github.com/0xsj/canopy-backend/internal/organization/service"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// Provider encapsulates organization context wiring.
type Provider struct {
	Service    *service.Service
	Handler    *handler.Handler
	MemberRepo *postgres.MemberRepository // exposed: workspace needs this for OrgMemberReader
}

// Wire creates the organization bounded context from infrastructure dependencies.
func Wire(db *database.DB, users service.UserReader, pub events.Publisher, log logger.Logger) *Provider {
	dbtx := db.DBTX()
	orgRepo := postgres.NewOrgRepository(dbtx)
	memberRepo := postgres.NewMemberRepository(dbtx)
	teamRepo := postgres.NewTeamRepository(dbtx)

	svc := service.New(
		orgRepo, memberRepo, teamRepo,
		users,
		db,
		func(tx database.DBTX) domain.OrgRepository { return postgres.NewOrgRepository(tx) },
		func(tx database.DBTX) domain.MemberRepository { return postgres.NewMemberRepository(tx) },
		pub, log,
	)
	h := handler.NewHandler(svc, log)
	return &Provider{Service: svc, Handler: h, MemberRepo: memberRepo}
}
