package workspace

import (
	"github.com/0xsj/canopy-backend/internal/workspace/adapter/postgres"
	"github.com/0xsj/canopy-backend/internal/workspace/domain"
	handler "github.com/0xsj/canopy-backend/internal/workspace/interface/http/v1"
	"github.com/0xsj/canopy-backend/internal/workspace/service"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// Provider encapsulates workspace context wiring.
type Provider struct {
	Service       *service.Service
	Handler       *handler.Handler
	MemberRepo    *postgres.WorkspaceMemberRepository // exposed: 7 contexts need WorkspaceMemberReader
	LLMConfigRepo *postgres.LLMConfigRepository       // exposed: composition root needs it for resolver
	WorkspaceRepo *postgres.WorkspaceRepository       // exposed: deliverable context needs config reader
}

// Wire creates the workspace bounded context from infrastructure dependencies.
func Wire(db *database.DB, orgMembers service.OrgMemberReader, encryptor service.Encryptor, pub events.Publisher, log logger.Logger) *Provider {
	dbtx := db.DBTX()
	wsRepo := postgres.NewWorkspaceRepository(dbtx)
	memberRepo := postgres.NewWorkspaceMemberRepository(dbtx)
	llmConfigRepo := postgres.NewLLMConfigRepository(dbtx)

	svc := service.New(
		wsRepo, memberRepo, llmConfigRepo,
		orgMembers, encryptor,
		db,
		func(tx database.DBTX) domain.WorkspaceRepository { return postgres.NewWorkspaceRepository(tx) },
		func(tx database.DBTX) domain.WorkspaceMemberRepository {
			return postgres.NewWorkspaceMemberRepository(tx)
		},
		pub, log,
	)
	h := handler.NewHandler(svc, log)
	return &Provider{Service: svc, Handler: h, MemberRepo: memberRepo, LLMConfigRepo: llmConfigRepo, WorkspaceRepo: wsRepo}
}
