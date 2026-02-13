package postgres

import (
	"context"

	"github.com/0xsj/canopy-backend/internal/workspace/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/workspace/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// WorkspaceMemberRepository implements domain.WorkspaceMemberRepository using Postgres via sqlc.
type WorkspaceMemberRepository struct {
	q *sqlc.Queries
}

// NewWorkspaceMemberRepository creates a new WorkspaceMemberRepository.
func NewWorkspaceMemberRepository(db database.DBTX) *WorkspaceMemberRepository {
	return &WorkspaceMemberRepository{q: sqlc.New(db)}
}

var _ domain.WorkspaceMemberRepository = (*WorkspaceMemberRepository)(nil)

func (r *WorkspaceMemberRepository) Add(ctx context.Context, member domain.WorkspaceMember) error {
	const op = "workspace: add member"
	return database.MapQueryError(r.q.AddWorkspaceMember(ctx, memberToAddParams(member)), op)
}

func (r *WorkspaceMemberRepository) Remove(ctx context.Context, workspaceID types.WorkspaceID, userID types.UserID) error {
	const op = "workspace: remove member"
	tag, err := r.q.RemoveWorkspaceMember(ctx, sqlc.RemoveWorkspaceMemberParams{
		WorkspaceID: workspaceID.String(),
		UserID:      userID.String(),
	})
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}

func (r *WorkspaceMemberRepository) FindByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]domain.WorkspaceMember, error) {
	const op = "workspace: find members by workspace"
	rows, err := r.q.FindWorkspaceMembersByWorkspace(ctx, workspaceID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return membersToDomain(rows), nil
}

func (r *WorkspaceMemberRepository) FindByUser(ctx context.Context, orgID types.OrgID, userID types.UserID) ([]domain.WorkspaceMember, error) {
	const op = "workspace: find memberships by user"
	rows, err := r.q.FindWorkspaceMembersByUser(ctx, sqlc.FindWorkspaceMembersByUserParams{
		OrgID:  orgID.String(),
		UserID: userID.String(),
	})
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return membersToDomain(rows), nil
}

func (r *WorkspaceMemberRepository) FindMember(ctx context.Context, workspaceID types.WorkspaceID, userID types.UserID) (domain.WorkspaceMember, error) {
	const op = "workspace: find member"
	row, err := r.q.FindWorkspaceMember(ctx, sqlc.FindWorkspaceMemberParams{
		WorkspaceID: workspaceID.String(),
		UserID:      userID.String(),
	})
	if err != nil {
		return domain.WorkspaceMember{}, database.MapQueryError(err, op)
	}
	return memberToDomain(row), nil
}

func (r *WorkspaceMemberRepository) UpdateRole(ctx context.Context, member domain.WorkspaceMember) error {
	const op = "workspace: update member role"
	tag, err := r.q.UpdateWorkspaceMemberRole(ctx, sqlc.UpdateWorkspaceMemberRoleParams{
		WorkspaceID: member.WorkspaceID().String(),
		UserID:      member.UserID().String(),
		Role:        string(member.Role()),
	})
	if err != nil {
		return database.MapQueryError(err, op)
	}
	if tag.RowsAffected() == 0 {
		return canopyerr.Wrap(canopyerr.ErrNotFound, op)
	}
	return nil
}
