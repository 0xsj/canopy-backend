package service

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	orgdomain "github.com/0xsj/canopy-backend/internal/organization/domain"
	"github.com/0xsj/canopy-backend/internal/workspace/domain"
	"github.com/0xsj/canopy-backend/pkg/auth"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// OrgMemberReader is a cross-context read port to validate org membership.
type OrgMemberReader interface {
	FindMember(ctx context.Context, orgID types.OrgID, userID types.UserID) (orgdomain.OrgMember, error)
}

// Service implements the workspace application logic.
type Service struct {
	workspaces domain.WorkspaceRepository
	members    domain.WorkspaceMemberRepository
	orgMembers OrgMemberReader
	db         *database.DB
	newWsRepo  func(database.DBTX) domain.WorkspaceRepository
	newMemRepo func(database.DBTX) domain.WorkspaceMemberRepository
	pub        events.Publisher
	log        logger.Logger
}

// New creates a new workspace service.
func New(
	workspaces domain.WorkspaceRepository,
	members domain.WorkspaceMemberRepository,
	orgMembers OrgMemberReader,
	db *database.DB,
	newWsRepo func(database.DBTX) domain.WorkspaceRepository,
	newMemRepo func(database.DBTX) domain.WorkspaceMemberRepository,
	pub events.Publisher,
	log logger.Logger,
) *Service {
	return &Service{
		workspaces: workspaces,
		members:    members,
		orgMembers: orgMembers,
		db:         db,
		newWsRepo:  newWsRepo,
		newMemRepo: newMemRepo,
		pub:        pub,
		log:        log,
	}
}

// CreateWorkspace creates a workspace and adds the caller as the owner member atomically.
func (s *Service) CreateWorkspace(ctx context.Context, orgID types.OrgID, name, description string) (domain.Workspace, error) {
	const op = "workspace: create"

	callerID, err := s.authenticatedUserID(ctx)
	if err != nil {
		return domain.Workspace{}, canopyerr.Wrap(err, op)
	}

	// Verify org membership.
	if _, err := s.orgMembers.FindMember(ctx, orgID, callerID); err != nil {
		if canopyerr.GetKind(err) == canopyerr.KindNotFound {
			return domain.Workspace{}, canopyerr.Wrap(canopyerr.ErrUnauthorized, op)
		}
		return domain.Workspace{}, canopyerr.Wrap(err, op)
	}

	ws, err := domain.NewWorkspace(orgID, name, description)
	if err != nil {
		return domain.Workspace{}, canopyerr.Wrap(err, op)
	}

	owner, err := domain.NewWorkspaceMember(ws.ID(), callerID, domain.RoleLoreKeeper)
	if err != nil {
		return domain.Workspace{}, canopyerr.Wrap(err, op)
	}

	if err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		txWs := s.newWsRepo(tx)
		txMem := s.newMemRepo(tx)
		if err := txWs.Create(ctx, ws); err != nil {
			return err
		}
		return txMem.Add(ctx, owner)
	}); err != nil {
		return domain.Workspace{}, canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectWorkspaceCreated, ws.ID().String(), domain.WorkspaceCreatedData{
		WorkspaceID: ws.ID().String(),
		OrgID:       orgID.String(),
		Name:        ws.Name(),
		Timestamp:   time.Now().UTC(),
	})

	s.log.Info("workspace created",
		logger.String("workspace_id", ws.ID().String()),
		logger.String("name", name),
	)

	return ws, nil
}

// FindByID returns a workspace by ID.
func (s *Service) FindByID(ctx context.Context, id types.WorkspaceID) (domain.Workspace, error) {
	const op = "workspace: find by id"
	ws, err := s.workspaces.FindByID(ctx, id)
	if err != nil {
		return domain.Workspace{}, canopyerr.Wrap(err, op)
	}
	return ws, nil
}

// FindByOrg returns all workspaces in an organization.
func (s *Service) FindByOrg(ctx context.Context, orgID types.OrgID) ([]domain.Workspace, error) {
	const op = "workspace: find by org"
	ws, err := s.workspaces.FindByOrg(ctx, orgID)
	if err != nil {
		return nil, canopyerr.Wrap(err, op)
	}
	return ws, nil
}

// JoinWorkspace adds the caller as a participant. Must be an org member.
func (s *Service) JoinWorkspace(ctx context.Context, workspaceID types.WorkspaceID) error {
	const op = "workspace: join"

	callerID, err := s.authenticatedUserID(ctx)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	ws, err := s.workspaces.FindByID(ctx, workspaceID)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	if _, err := s.orgMembers.FindMember(ctx, ws.OrgID(), callerID); err != nil {
		if canopyerr.GetKind(err) == canopyerr.KindNotFound {
			return canopyerr.Wrap(canopyerr.ErrUnauthorized, op)
		}
		return canopyerr.Wrap(err, op)
	}

	member, err := domain.NewWorkspaceMember(workspaceID, callerID, domain.RoleParticipant)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := s.members.Add(ctx, member); err != nil {
		return canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectMemberJoined, workspaceID.String(), domain.MemberJoinedData{
		WorkspaceID: workspaceID.String(),
		UserID:      callerID.String(),
		Role:        string(domain.RoleParticipant),
		Timestamp:   time.Now().UTC(),
	})

	return nil
}

// LeaveWorkspace removes the caller from a workspace.
func (s *Service) LeaveWorkspace(ctx context.Context, workspaceID types.WorkspaceID) error {
	const op = "workspace: leave"

	callerID, err := s.authenticatedUserID(ctx)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := s.members.Remove(ctx, workspaceID, callerID); err != nil {
		return canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectMemberLeft, workspaceID.String(), domain.MemberLeftData{
		WorkspaceID: workspaceID.String(),
		UserID:      callerID.String(),
		Timestamp:   time.Now().UTC(),
	})

	return nil
}

// UpdateRole changes a workspace member's role. Caller must be lore keeper.
func (s *Service) UpdateRole(ctx context.Context, workspaceID types.WorkspaceID, userID types.UserID, newRole domain.WorkspaceRole) error {
	const op = "workspace: update role"

	if err := s.requireLoreKeeper(ctx, workspaceID); err != nil {
		return canopyerr.Wrap(err, op)
	}

	member, err := s.members.FindMember(ctx, workspaceID, userID)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := member.ChangeRole(newRole); err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := s.members.UpdateRole(ctx, member); err != nil {
		return canopyerr.Wrap(err, op)
	}

	return nil
}

// UpdateConfig updates the workspace configuration. Caller must be lore keeper.
func (s *Service) UpdateConfig(ctx context.Context, workspaceID types.WorkspaceID, mode domain.LoreKeeperMode, loreKeeperID types.UserID) error {
	const op = "workspace: update config"

	if err := s.requireLoreKeeper(ctx, workspaceID); err != nil {
		return canopyerr.Wrap(err, op)
	}

	ws, err := s.workspaces.FindByID(ctx, workspaceID)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := ws.SetLoreKeeper(mode, loreKeeperID); err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := s.workspaces.Update(ctx, ws); err != nil {
		return canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectWorkspaceConfigUpdated, workspaceID.String(), domain.WorkspaceConfigUpdatedData{
		WorkspaceID:    workspaceID.String(),
		LoreKeeperMode: string(mode),
		LoreKeeperID:   loreKeeperID.String(),
		Timestamp:      time.Now().UTC(),
	})

	return nil
}

// TransitionPhase advances the workspace to the next phase. Caller must be lore keeper.
func (s *Service) TransitionPhase(ctx context.Context, workspaceID types.WorkspaceID, target domain.Phase) error {
	const op = "workspace: transition phase"

	if err := s.requireLoreKeeper(ctx, workspaceID); err != nil {
		return canopyerr.Wrap(err, op)
	}

	ws, err := s.workspaces.FindByID(ctx, workspaceID)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	previousPhase := ws.Phase()
	if err := ws.TransitionPhase(target); err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := s.workspaces.Update(ctx, ws); err != nil {
		return canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectPhaseTransitioned, workspaceID.String(), domain.PhaseTransitionedData{
		WorkspaceID:   workspaceID.String(),
		PreviousPhase: string(previousPhase),
		NewPhase:      string(target),
		Timestamp:     time.Now().UTC(),
	})

	return nil
}

// --- Auth Helpers ---

func (s *Service) authenticatedUserID(ctx context.Context) (types.UserID, error) {
	claims, ok := auth.FromClaims(ctx)
	if !ok {
		return types.UserID{}, canopyerr.ErrUnauthenticated
	}
	return types.UserIDFrom(claims.Subject), nil
}

func (s *Service) requireLoreKeeper(ctx context.Context, workspaceID types.WorkspaceID) error {
	callerID, err := s.authenticatedUserID(ctx)
	if err != nil {
		return err
	}
	member, err := s.members.FindMember(ctx, workspaceID, callerID)
	if err != nil {
		if canopyerr.GetKind(err) == canopyerr.KindNotFound {
			return canopyerr.ErrUnauthorized
		}
		return err
	}
	if member.Role() != domain.RoleLoreKeeper {
		return canopyerr.ErrUnauthorized
	}
	return nil
}

// RequireMember validates workspace membership for the authenticated caller.
// Exported for use by other contexts' services as a cross-context check.
func (s *Service) RequireMember(ctx context.Context, workspaceID types.WorkspaceID) (types.UserID, error) {
	callerID, err := s.authenticatedUserID(ctx)
	if err != nil {
		return types.UserID{}, err
	}
	if _, err := s.members.FindMember(ctx, workspaceID, callerID); err != nil {
		if canopyerr.GetKind(err) == canopyerr.KindNotFound {
			return types.UserID{}, canopyerr.ErrUnauthorized
		}
		return types.UserID{}, err
	}
	return callerID, nil
}

// --- Event Publishing ---

func (s *Service) publish(ctx context.Context, eventType, workspaceID string, data any) {
	event, err := events.New(eventType, workspaceID, data)
	if err != nil {
		s.log.Error("event creation failed", logger.String("type", eventType), logger.Err(err))
		return
	}
	event.Subject = events.BuildSubject(workspaceID, "workspace", eventType)
	if pubErr := s.pub.Publish(ctx, event); pubErr != nil {
		s.log.Error("event publish failed", logger.String("type", eventType), logger.Err(pubErr))
	}
}
