package postgres

import (
	"context"
	"time"

	"github.com/0xsj/canopy-backend/internal/organization/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// MemberRepository implements domain.MemberRepository using Postgres.
type MemberRepository struct {
	db database.DBTX
}

// NewMemberRepository creates a new MemberRepository.
func NewMemberRepository(db database.DBTX) *MemberRepository {
	return &MemberRepository{db: db}
}

var _ domain.MemberRepository = (*MemberRepository)(nil)

func (r *MemberRepository) Add(ctx context.Context, member domain.OrgMember) error {
	const op = "organization: add member"
	const query = `
		INSERT INTO org_members (org_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, $4)`

	_, err := r.db.Exec(ctx, query,
		member.OrgID().String(),
		member.UserID().String(),
		string(member.Role()),
		member.JoinedAt().Time(),
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return nil
}

func (r *MemberRepository) Remove(ctx context.Context, orgID types.OrgID, userID types.UserID) error {
	const op = "organization: remove member"
	const query = `DELETE FROM org_members WHERE org_id = $1 AND user_id = $2`

	tag, err := r.db.Exec(ctx, query, orgID.String(), userID.String())
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}

func (r *MemberRepository) FindByOrg(ctx context.Context, orgID types.OrgID) ([]domain.OrgMember, error) {
	const op = "organization: find members by org"
	const query = `
		SELECT org_id, user_id, role, joined_at
		FROM org_members WHERE org_id = $1 ORDER BY joined_at`

	return r.queryMembers(ctx, query, op, orgID.String())
}

func (r *MemberRepository) FindByUser(ctx context.Context, userID types.UserID) ([]domain.OrgMember, error) {
	const op = "organization: find memberships by user"
	const query = `
		SELECT org_id, user_id, role, joined_at
		FROM org_members WHERE user_id = $1 ORDER BY joined_at`

	return r.queryMembers(ctx, query, op, userID.String())
}

func (r *MemberRepository) FindMember(ctx context.Context, orgID types.OrgID, userID types.UserID) (domain.OrgMember, error) {
	const op = "organization: find member"
	const query = `
		SELECT org_id, user_id, role, joined_at
		FROM org_members WHERE org_id = $1 AND user_id = $2`

	return scanOrgMember(r.db.QueryRow(ctx, query, orgID.String(), userID.String()), op)
}

func (r *MemberRepository) UpdateRole(ctx context.Context, member domain.OrgMember) error {
	const op = "organization: update member role"
	const query = `
		UPDATE org_members SET role = $3
		WHERE org_id = $1 AND user_id = $2`

	tag, err := r.db.Exec(ctx, query,
		member.OrgID().String(),
		member.UserID().String(),
		string(member.Role()),
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}

func (r *MemberRepository) CountByRole(ctx context.Context, orgID types.OrgID, role domain.Role) (int, error) {
	const op = "organization: count members by role"
	const query = `SELECT COUNT(*) FROM org_members WHERE org_id = $1 AND role = $2`

	var count int
	if err := r.db.QueryRow(ctx, query, orgID.String(), string(role)).Scan(&count); err != nil {
		return 0, database.MapQueryError(err, op)
	}
	return count, nil
}

// --- helpers ---

func (r *MemberRepository) queryMembers(ctx context.Context, query, op string, args ...any) ([]domain.OrgMember, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	defer rows.Close()

	var members []domain.OrgMember
	for rows.Next() {
		m, err := scanOrgMember(rows, op)
		if err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func scanOrgMember(row rowScanner, op string) (domain.OrgMember, error) {
	var (
		rawOrgID  string
		rawUserID string
		role      string
		joinedAt  time.Time
	)

	if err := row.Scan(&rawOrgID, &rawUserID, &role, &joinedAt); err != nil {
		return domain.OrgMember{}, database.MapQueryError(err, op)
	}

	return domain.ReconstructOrgMember(
		types.OrgIDFrom(rawOrgID),
		types.UserIDFrom(rawUserID),
		domain.Role(role),
		types.TimestampFrom(joinedAt),
	), nil
}
