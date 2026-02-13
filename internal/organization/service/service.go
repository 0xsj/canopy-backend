package service

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/0xsj/canopy-backend/internal/organization/domain"
	"github.com/0xsj/canopy-backend/pkg/auth"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// UserReader is a cross-context read port for identity data.
type UserReader interface {
	FindByID(ctx context.Context, id types.UserID) (UserInfo, error)
}

// UserInfo is a projection of the identity user, carrying only what
// the organization context needs.
type UserInfo struct {
	ID          types.UserID
	DisplayName string
	Email       string
}

// Service implements the organization application logic.
type Service struct {
	orgs    domain.OrgRepository
	members domain.MemberRepository
	teams   domain.TeamRepository
	users   UserReader
	db      *database.DB
	// Factory functions for creating tx-scoped repo instances.
	newOrgRepo    func(database.DBTX) domain.OrgRepository
	newMemberRepo func(database.DBTX) domain.MemberRepository
	pub           events.Publisher
	log           logger.Logger
}

// New creates a new organization service.
func New(
	orgs domain.OrgRepository,
	members domain.MemberRepository,
	teams domain.TeamRepository,
	users UserReader,
	db *database.DB,
	newOrgRepo func(database.DBTX) domain.OrgRepository,
	newMemberRepo func(database.DBTX) domain.MemberRepository,
	pub events.Publisher,
	log logger.Logger,
) *Service {
	return &Service{
		orgs:          orgs,
		members:       members,
		teams:         teams,
		users:         users,
		db:            db,
		newOrgRepo:    newOrgRepo,
		newMemberRepo: newMemberRepo,
		pub:           pub,
		log:           log,
	}
}

// --- Organization CRUD ---

// CreateOrg creates an organization and adds the caller as the owner member atomically.
func (s *Service) CreateOrg(ctx context.Context, name, slug string) (domain.Organization, error) {
	const op = "organization: create org"

	callerID, err := s.authenticatedUserID(ctx)
	if err != nil {
		return domain.Organization{}, canopyerr.Wrap(err, op)
	}

	org, err := domain.NewOrganization(name, slug, callerID)
	if err != nil {
		return domain.Organization{}, canopyerr.Wrap(err, op)
	}

	owner, err := domain.NewOrgMember(org.ID(), callerID, domain.RoleOwner)
	if err != nil {
		return domain.Organization{}, canopyerr.Wrap(err, op)
	}

	if err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		txOrgs := s.newOrgRepo(tx)
		txMembers := s.newMemberRepo(tx)
		if err := txOrgs.Create(ctx, org); err != nil {
			return err
		}
		return txMembers.Add(ctx, owner)
	}); err != nil {
		return domain.Organization{}, canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectOrgCreated, "", domain.OrgCreatedData{
		OrgID:     org.ID().String(),
		Name:      org.Name(),
		Slug:      org.Slug(),
		Personal:  org.IsPersonal(),
		OwnerID:   callerID.String(),
		Timestamp: time.Now().UTC(),
	})

	s.log.Info("org created",
		logger.String("org_id", org.ID().String()),
		logger.String("name", name),
	)

	return org, nil
}

// FindOrgByID returns an organization by ID.
func (s *Service) FindOrgByID(ctx context.Context, id types.OrgID) (domain.Organization, error) {
	const op = "organization: find org by id"
	org, err := s.orgs.FindByID(ctx, id)
	if err != nil {
		return domain.Organization{}, canopyerr.Wrap(err, op)
	}
	return org, nil
}

// FindOrgBySlug returns an organization by slug.
func (s *Service) FindOrgBySlug(ctx context.Context, slug string) (domain.Organization, error) {
	const op = "organization: find org by slug"
	org, err := s.orgs.FindBySlug(ctx, slug)
	if err != nil {
		return domain.Organization{}, canopyerr.Wrap(err, op)
	}
	return org, nil
}

// --- Member Management ---

// AddMember adds a user to an organization. Caller must be owner or admin.
func (s *Service) AddMember(ctx context.Context, orgID types.OrgID, userID types.UserID, role domain.Role) error {
	const op = "organization: add member"

	if err := s.requireRole(ctx, orgID, domain.RoleAdmin); err != nil {
		return canopyerr.Wrap(err, op)
	}

	member, err := domain.NewOrgMember(orgID, userID, role)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := s.members.Add(ctx, member); err != nil {
		return canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectMemberAdded, "", domain.MemberAddedData{
		OrgID:     orgID.String(),
		UserID:    userID.String(),
		Role:      string(role),
		Timestamp: time.Now().UTC(),
	})

	return nil
}

// RemoveMember removes a user from an organization. Caller must be owner or admin.
func (s *Service) RemoveMember(ctx context.Context, orgID types.OrgID, userID types.UserID) error {
	const op = "organization: remove member"

	if err := s.requireRole(ctx, orgID, domain.RoleAdmin); err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := s.members.Remove(ctx, orgID, userID); err != nil {
		return canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectMemberRemoved, "", domain.MemberRemovedData{
		OrgID:     orgID.String(),
		UserID:    userID.String(),
		Timestamp: time.Now().UTC(),
	})

	return nil
}

// UpdateMemberRole changes a member's role. Caller must be owner or admin.
func (s *Service) UpdateMemberRole(ctx context.Context, orgID types.OrgID, userID types.UserID, newRole domain.Role) error {
	const op = "organization: update member role"

	if err := s.requireRole(ctx, orgID, domain.RoleAdmin); err != nil {
		return canopyerr.Wrap(err, op)
	}

	member, err := s.members.FindMember(ctx, orgID, userID)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	previousRole := member.Role()
	if err := member.ChangeRole(newRole); err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := s.members.UpdateRole(ctx, member); err != nil {
		return canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectMemberRoleChanged, "", domain.MemberRoleChangedData{
		OrgID:        orgID.String(),
		UserID:       userID.String(),
		PreviousRole: string(previousRole),
		NewRole:      string(newRole),
		Timestamp:    time.Now().UTC(),
	})

	return nil
}

// ListMembers returns all members of an organization. Caller must be a member.
func (s *Service) ListMembers(ctx context.Context, orgID types.OrgID) ([]domain.OrgMember, error) {
	const op = "organization: list members"

	if err := s.requireRole(ctx, orgID, domain.RoleMember); err != nil {
		return nil, canopyerr.Wrap(err, op)
	}

	members, err := s.members.FindByOrg(ctx, orgID)
	if err != nil {
		return nil, canopyerr.Wrap(err, op)
	}
	return members, nil
}

// --- Team Management ---

// CreateTeam creates a new team in an organization. Caller must be owner or admin.
func (s *Service) CreateTeam(ctx context.Context, orgID types.OrgID, name string) (domain.Team, error) {
	const op = "organization: create team"

	if err := s.requireRole(ctx, orgID, domain.RoleAdmin); err != nil {
		return domain.Team{}, canopyerr.Wrap(err, op)
	}

	team, err := domain.NewTeam(orgID, name)
	if err != nil {
		return domain.Team{}, canopyerr.Wrap(err, op)
	}

	if err := s.teams.Create(ctx, team); err != nil {
		return domain.Team{}, canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectTeamCreated, "", domain.TeamCreatedData{
		TeamID:    team.ID().String(),
		OrgID:     orgID.String(),
		Name:      team.Name(),
		Timestamp: time.Now().UTC(),
	})

	return team, nil
}

// AddTeamMember adds a user to a team. Caller must be owner or admin.
func (s *Service) AddTeamMember(ctx context.Context, orgID types.OrgID, teamID types.TeamID, userID types.UserID) error {
	const op = "organization: add team member"

	if err := s.requireRole(ctx, orgID, domain.RoleAdmin); err != nil {
		return canopyerr.Wrap(err, op)
	}

	member, err := domain.NewTeamMember(teamID, userID)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := s.teams.AddMember(ctx, member); err != nil {
		return canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectTeamMemberAdded, "", domain.TeamMemberAddedData{
		TeamID:    teamID.String(),
		OrgID:     orgID.String(),
		UserID:    userID.String(),
		Timestamp: time.Now().UTC(),
	})

	return nil
}

// RemoveTeamMember removes a user from a team. Caller must be owner or admin.
func (s *Service) RemoveTeamMember(ctx context.Context, orgID types.OrgID, teamID types.TeamID, userID types.UserID) error {
	const op = "organization: remove team member"

	if err := s.requireRole(ctx, orgID, domain.RoleAdmin); err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := s.teams.RemoveMember(ctx, teamID, userID); err != nil {
		return canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectTeamMemberRemoved, "", domain.TeamMemberRemovedData{
		TeamID:    teamID.String(),
		OrgID:     orgID.String(),
		UserID:    userID.String(),
		Timestamp: time.Now().UTC(),
	})

	return nil
}

// ListTeams returns all teams in an organization. Caller must be a member.
func (s *Service) ListTeams(ctx context.Context, orgID types.OrgID) ([]domain.Team, error) {
	const op = "organization: list teams"

	if err := s.requireRole(ctx, orgID, domain.RoleMember); err != nil {
		return nil, canopyerr.Wrap(err, op)
	}

	teams, err := s.teams.FindByOrg(ctx, orgID)
	if err != nil {
		return nil, canopyerr.Wrap(err, op)
	}
	return teams, nil
}

// ListTeamMembers returns all members of a team. Caller must be an org member.
func (s *Service) ListTeamMembers(ctx context.Context, orgID types.OrgID, teamID types.TeamID) ([]domain.TeamMember, error) {
	const op = "organization: list team members"

	if err := s.requireRole(ctx, orgID, domain.RoleMember); err != nil {
		return nil, canopyerr.Wrap(err, op)
	}

	members, err := s.teams.FindMembers(ctx, teamID)
	if err != nil {
		return nil, canopyerr.Wrap(err, op)
	}
	return members, nil
}

// --- Auth Helpers ---

func (s *Service) authenticatedUserID(ctx context.Context) (types.UserID, error) {
	claims, ok := auth.FromClaims(ctx)
	if !ok {
		return types.UserID{}, canopyerr.ErrUnauthenticated
	}
	return types.UserIDFrom(claims.Subject), nil
}

// requireRole validates that the authenticated caller has at least the given
// role in the organization. Owner satisfies admin; admin satisfies member.
func (s *Service) requireRole(ctx context.Context, orgID types.OrgID, minimum domain.Role) error {
	callerID, err := s.authenticatedUserID(ctx)
	if err != nil {
		return err
	}

	member, err := s.members.FindMember(ctx, orgID, callerID)
	if err != nil {
		if canopyerr.GetKind(err) == canopyerr.KindNotFound {
			return canopyerr.ErrUnauthorized
		}
		return err
	}

	if !roleSatisfies(member.Role(), minimum) {
		return canopyerr.ErrUnauthorized
	}
	return nil
}

func roleSatisfies(actual, minimum domain.Role) bool {
	hierarchy := map[domain.Role]int{
		domain.RoleOwner:  2,
		domain.RoleAdmin:  1,
		domain.RoleMember: 0,
	}
	return hierarchy[actual] >= hierarchy[minimum]
}

// --- Event Publishing ---

func (s *Service) publish(ctx context.Context, eventType, workspaceID string, data any) {
	event, err := events.New(eventType, workspaceID, data)
	if err != nil {
		s.log.Error("event creation failed", logger.String("type", eventType), logger.Err(err))
		return
	}
	event.Subject = events.BuildSubject(workspaceID, "organization", eventType)
	if pubErr := s.pub.Publish(ctx, event); pubErr != nil {
		s.log.Error("event publish failed", logger.String("type", eventType), logger.Err(pubErr))
	}
}
