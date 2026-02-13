package postgres

import (
	"context"

	"github.com/0xsj/canopy-backend/internal/organization/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/organization/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// MemberRepository implements domain.MemberRepository using Postgres via sqlc.
type MemberRepository struct {
	q *sqlc.Queries
}

// NewMemberRepository creates a new MemberRepository.
func NewMemberRepository(db database.DBTX) *MemberRepository {
	return &MemberRepository{q: sqlc.New(db)}
}

var _ domain.MemberRepository = (*MemberRepository)(nil)

func (r *MemberRepository) Add(ctx context.Context, member domain.OrgMember) error {
	const op = "organization: add member"
	return database.MapQueryError(r.q.AddOrgMember(ctx, orgMemberToAddParams(member)), op)
}

func (r *MemberRepository) Remove(ctx context.Context, orgID types.OrgID, userID types.UserID) error {
	const op = "organization: remove member"
	tag, err := r.q.RemoveOrgMember(ctx, sqlc.RemoveOrgMemberParams{
		OrgID:  orgID.String(),
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

func (r *MemberRepository) FindByOrg(ctx context.Context, orgID types.OrgID) ([]domain.OrgMember, error) {
	const op = "organization: find members by org"
	rows, err := r.q.FindOrgMembersByOrg(ctx, orgID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return orgMembersToDomain(rows), nil
}

func (r *MemberRepository) FindByUser(ctx context.Context, userID types.UserID) ([]domain.OrgMember, error) {
	const op = "organization: find memberships by user"
	rows, err := r.q.FindOrgMembersByUser(ctx, userID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return orgMembersToDomain(rows), nil
}

func (r *MemberRepository) FindMember(ctx context.Context, orgID types.OrgID, userID types.UserID) (domain.OrgMember, error) {
	const op = "organization: find member"
	row, err := r.q.FindOrgMember(ctx, sqlc.FindOrgMemberParams{
		OrgID:  orgID.String(),
		UserID: userID.String(),
	})
	if err != nil {
		return domain.OrgMember{}, database.MapQueryError(err, op)
	}
	return orgMemberToDomain(row), nil
}

func (r *MemberRepository) UpdateRole(ctx context.Context, member domain.OrgMember) error {
	const op = "organization: update member role"
	tag, err := r.q.UpdateOrgMemberRole(ctx, sqlc.UpdateOrgMemberRoleParams{
		OrgID:  member.OrgID().String(),
		UserID: member.UserID().String(),
		Role:   string(member.Role()),
	})
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
	count, err := r.q.CountOrgMembersByRole(ctx, sqlc.CountOrgMembersByRoleParams{
		OrgID: orgID.String(),
		Role:  string(role),
	})
	if err != nil {
		return 0, database.MapQueryError(err, op)
	}
	return int(count), nil
}
