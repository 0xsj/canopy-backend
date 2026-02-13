package postgres

import (
	"context"
	"time"

	"github.com/0xsj/canopy-backend/internal/organization/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// TeamRepository implements domain.TeamRepository using Postgres.
type TeamRepository struct {
	db database.DBTX
}

// NewTeamRepository creates a new TeamRepository.
func NewTeamRepository(db database.DBTX) *TeamRepository {
	return &TeamRepository{db: db}
}

var _ domain.TeamRepository = (*TeamRepository)(nil)

func (r *TeamRepository) Create(ctx context.Context, team domain.Team) error {
	const op = "organization: create team"
	const query = `
		INSERT INTO teams (id, org_id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	ts := team.Timestamps()
	_, err := r.db.Exec(ctx, query,
		team.ID().String(),
		team.OrgID().String(),
		team.Name(),
		nullableString(team.Description()),
		ts.CreatedAt.Time(),
		tsUpdatedAt(ts),
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return nil
}

func (r *TeamRepository) FindByID(ctx context.Context, id types.TeamID) (domain.Team, error) {
	const op = "organization: find team by id"
	const query = `
		SELECT id, org_id, name, description, created_at, updated_at
		FROM teams WHERE id = $1`

	return scanTeam(r.db.QueryRow(ctx, query, id.String()), op)
}

func (r *TeamRepository) FindByOrg(ctx context.Context, orgID types.OrgID) ([]domain.Team, error) {
	const op = "organization: find teams by org"
	const query = `
		SELECT id, org_id, name, description, created_at, updated_at
		FROM teams WHERE org_id = $1 ORDER BY name`

	rows, err := r.db.Query(ctx, query, orgID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	defer rows.Close()

	var teams []domain.Team
	for rows.Next() {
		t, err := scanTeam(rows, op)
		if err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, rows.Err()
}

func (r *TeamRepository) Update(ctx context.Context, team domain.Team) error {
	const op = "organization: update team"
	const query = `
		UPDATE teams SET name = $2, description = $3, updated_at = $4
		WHERE id = $1`

	ts := team.Timestamps()
	tag, err := r.db.Exec(ctx, query,
		team.ID().String(),
		team.Name(),
		nullableString(team.Description()),
		tsUpdatedAt(ts),
	)
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
	const query = `DELETE FROM teams WHERE id = $1`

	tag, err := r.db.Exec(ctx, query, id.String())
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
	const query = `
		INSERT INTO team_members (team_id, user_id, joined_at)
		VALUES ($1, $2, $3)`

	_, err := r.db.Exec(ctx, query,
		member.TeamID().String(),
		member.UserID().String(),
		member.JoinedAt().Time(),
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return nil
}

func (r *TeamRepository) RemoveMember(ctx context.Context, teamID types.TeamID, userID types.UserID) error {
	const op = "organization: remove team member"
	const query = `DELETE FROM team_members WHERE team_id = $1 AND user_id = $2`

	tag, err := r.db.Exec(ctx, query, teamID.String(), userID.String())
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
	const query = `
		SELECT team_id, user_id, joined_at
		FROM team_members WHERE team_id = $1 ORDER BY joined_at`

	rows, err := r.db.Query(ctx, query, teamID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	defer rows.Close()

	var members []domain.TeamMember
	for rows.Next() {
		m, err := scanTeamMember(rows, op)
		if err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *TeamRepository) FindTeamsByUser(ctx context.Context, orgID types.OrgID, userID types.UserID) ([]domain.Team, error) {
	const op = "organization: find teams by user"
	const query = `
		SELECT t.id, t.org_id, t.name, t.description, t.created_at, t.updated_at
		FROM teams t
		JOIN team_members tm ON t.id = tm.team_id
		WHERE t.org_id = $1 AND tm.user_id = $2
		ORDER BY t.name`

	rows, err := r.db.Query(ctx, query, orgID.String(), userID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	defer rows.Close()

	var teams []domain.Team
	for rows.Next() {
		t, err := scanTeam(rows, op)
		if err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, rows.Err()
}

// --- helpers ---

func scanTeam(row rowScanner, op string) (domain.Team, error) {
	var (
		rawID       string
		rawOrgID    string
		name        string
		description *string
		createdAt   time.Time
		updatedAt   time.Time
	)

	if err := row.Scan(&rawID, &rawOrgID, &name, &description, &createdAt, &updatedAt); err != nil {
		return domain.Team{}, database.MapQueryError(err, op)
	}

	updatedTS := types.TimestampFrom(updatedAt)
	return domain.ReconstructTeam(
		types.TeamIDFrom(rawID),
		types.OrgIDFrom(rawOrgID),
		name,
		derefString(description),
		types.Timestamps{
			CreatedAt: types.TimestampFrom(createdAt),
			UpdatedAt: &updatedTS,
		},
	), nil
}

func scanTeamMember(row rowScanner, op string) (domain.TeamMember, error) {
	var (
		rawTeamID string
		rawUserID string
		joinedAt  time.Time
	)

	if err := row.Scan(&rawTeamID, &rawUserID, &joinedAt); err != nil {
		return domain.TeamMember{}, database.MapQueryError(err, op)
	}

	return domain.ReconstructTeamMember(
		types.TeamIDFrom(rawTeamID),
		types.UserIDFrom(rawUserID),
		types.TimestampFrom(joinedAt),
	), nil
}
