package postgres

import (
	"context"
	"time"

	"github.com/0xsj/canopy-backend/internal/workspace/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// WorkspaceMemberRepository implements domain.WorkspaceMemberRepository using Postgres.
type WorkspaceMemberRepository struct {
	db database.DBTX
}

// NewWorkspaceMemberRepository creates a new WorkspaceMemberRepository.
func NewWorkspaceMemberRepository(db database.DBTX) *WorkspaceMemberRepository {
	return &WorkspaceMemberRepository{db: db}
}

var _ domain.WorkspaceMemberRepository = (*WorkspaceMemberRepository)(nil)

func (r *WorkspaceMemberRepository) Add(ctx context.Context, member domain.WorkspaceMember) error {
	const op = "workspace: add member"
	const query = `
		INSERT INTO workspace_members (workspace_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, $4)`

	_, err := r.db.Exec(ctx, query,
		member.WorkspaceID().String(),
		member.UserID().String(),
		string(member.Role()),
		member.JoinedAt().Time(),
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return nil
}

func (r *WorkspaceMemberRepository) Remove(ctx context.Context, workspaceID types.WorkspaceID, userID types.UserID) error {
	const op = "workspace: remove member"
	const query = `DELETE FROM workspace_members WHERE workspace_id = $1 AND user_id = $2`

	tag, err := r.db.Exec(ctx, query, workspaceID.String(), userID.String())
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
	const query = `
		SELECT workspace_id, user_id, role, joined_at
		FROM workspace_members WHERE workspace_id = $1 ORDER BY joined_at`

	return r.queryMembers(ctx, query, op, workspaceID.String())
}

func (r *WorkspaceMemberRepository) FindByUser(ctx context.Context, orgID types.OrgID, userID types.UserID) ([]domain.WorkspaceMember, error) {
	const op = "workspace: find memberships by user"
	const query = `
		SELECT wm.workspace_id, wm.user_id, wm.role, wm.joined_at
		FROM workspace_members wm
		JOIN workspaces w ON w.id = wm.workspace_id
		WHERE w.org_id = $1 AND wm.user_id = $2
		ORDER BY wm.joined_at`

	return r.queryMembers(ctx, query, op, orgID.String(), userID.String())
}

func (r *WorkspaceMemberRepository) FindMember(ctx context.Context, workspaceID types.WorkspaceID, userID types.UserID) (domain.WorkspaceMember, error) {
	const op = "workspace: find member"
	const query = `
		SELECT workspace_id, user_id, role, joined_at
		FROM workspace_members WHERE workspace_id = $1 AND user_id = $2`

	return scanWorkspaceMember(r.db.QueryRow(ctx, query, workspaceID.String(), userID.String()), op)
}

func (r *WorkspaceMemberRepository) UpdateRole(ctx context.Context, member domain.WorkspaceMember) error {
	const op = "workspace: update member role"
	const query = `
		UPDATE workspace_members SET role = $3
		WHERE workspace_id = $1 AND user_id = $2`

	tag, err := r.db.Exec(ctx, query,
		member.WorkspaceID().String(),
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

// --- helpers ---

func (r *WorkspaceMemberRepository) queryMembers(ctx context.Context, query, op string, args ...any) ([]domain.WorkspaceMember, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	defer rows.Close()

	var members []domain.WorkspaceMember
	for rows.Next() {
		m, err := scanWorkspaceMember(rows, op)
		if err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func scanWorkspaceMember(row rowScanner, op string) (domain.WorkspaceMember, error) {
	var (
		rawWorkspaceID string
		rawUserID      string
		role           string
		joinedAt       time.Time
	)

	if err := row.Scan(&rawWorkspaceID, &rawUserID, &role, &joinedAt); err != nil {
		return domain.WorkspaceMember{}, database.MapQueryError(err, op)
	}

	return domain.ReconstructWorkspaceMember(
		types.WorkspaceIDFrom(rawWorkspaceID),
		types.UserIDFrom(rawUserID),
		domain.WorkspaceRole(role),
		types.TimestampFrom(joinedAt),
	), nil
}
