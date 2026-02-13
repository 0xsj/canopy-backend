package postgres

import (
	"context"

	"github.com/0xsj/canopy-backend/internal/organization/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/organization/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// TeamRepository implements domain.TeamRepository using Postgres via sqlc.
type TeamRepository struct {
	q *sqlc.Queries
}

// NewTeamRepository creates a new TeamRepository.
func NewTeamRepository(db database.DBTX) *TeamRepository {
	return &TeamRepository{q: sqlc.New(db)}
}

var _ domain.TeamRepository = (*TeamRepository)(nil)

func (r *TeamRepository) Create(ctx context.Context, team domain.Team) error {
	const op = "organization: create team"
	return database.MapQueryError(r.q.CreateTeam(ctx, teamToCreateParams(team)), op)
}

func (r *TeamRepository) FindByID(ctx context.Context, id types.TeamID) (domain.Team, error) {
	const op = "organization: find team by id"
	row, err := r.q.FindTeamByID(ctx, id.String())
	if err != nil {
		return domain.Team{}, database.MapQueryError(err, op)
	}
	return teamToDomain(row), nil
}

func (r *TeamRepository) FindByOrg(ctx context.Context, orgID types.OrgID) ([]domain.Team, error) {
	const op = "organization: find teams by org"
	rows, err := r.q.FindTeamsByOrg(ctx, orgID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return teamsToDomain(rows), nil
}

func (r *TeamRepository) Update(ctx context.Context, team domain.Team) error {
	const op = "organization: update team"
	tag, err := r.q.UpdateTeam(ctx, teamToUpdateParams(team))
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}

func (r *TeamRepository) Delete(ctx context.Context, id types.TeamID) error {
	const op = "organization: delete team"
	tag, err := r.q.DeleteTeam(ctx, id.String())
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}

func (r *TeamRepository) AddMember(ctx context.Context, member domain.TeamMember) error {
	const op = "organization: add team member"
	return database.MapQueryError(r.q.AddTeamMember(ctx, teamMemberToAddParams(member)), op)
}

func (r *TeamRepository) RemoveMember(ctx context.Context, teamID types.TeamID, userID types.UserID) error {
	const op = "organization: remove team member"
	tag, err := r.q.RemoveTeamMember(ctx, sqlc.RemoveTeamMemberParams{
		TeamID: teamID.String(),
		UserID: userID.String(),
	})
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}

func (r *TeamRepository) FindMembers(ctx context.Context, teamID types.TeamID) ([]domain.TeamMember, error) {
	const op = "organization: find team members"
	rows, err := r.q.FindTeamMembers(ctx, teamID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return teamMembersToDomain(rows), nil
}

func (r *TeamRepository) FindTeamsByUser(ctx context.Context, orgID types.OrgID, userID types.UserID) ([]domain.Team, error) {
	const op = "organization: find teams by user"
	rows, err := r.q.FindTeamsByUser(ctx, sqlc.FindTeamsByUserParams{
		OrgID:  orgID.String(),
		UserID: userID.String(),
	})
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return teamsToDomain(rows), nil
}
