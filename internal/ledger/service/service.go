package service

import (
	"context"

	"github.com/0xsj/canopy-backend/internal/ledger/domain"
	wsdomain "github.com/0xsj/canopy-backend/internal/workspace/domain"
	"github.com/0xsj/canopy-backend/pkg/auth"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// WorkspaceMemberReader is a cross-context read port for workspace membership.
type WorkspaceMemberReader interface {
	FindMember(ctx context.Context, workspaceID types.WorkspaceID, userID types.UserID) (wsdomain.WorkspaceMember, error)
}

// Service implements the ledger application logic.
// The ledger is a sink — it consumes events and records entries
// but publishes no events of its own.
type Service struct {
	repo      domain.LedgerRepository
	wsMembers WorkspaceMemberReader
	log       logger.Logger
}

// New creates a new ledger service.
func New(repo domain.LedgerRepository, wsMembers WorkspaceMemberReader, log logger.Logger) *Service {
	return &Service{repo: repo, wsMembers: wsMembers, log: log}
}

// AppendSystem records a system-level audit entry from a domain event.
func (s *Service) AppendSystem(ctx context.Context, eventSubject string, eventData map[string]any, sourceContext string) error {
	const op = "ledger: append system"

	entry, err := domain.NewSystemEntry(eventSubject, eventData, sourceContext)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := s.repo.AppendSystem(ctx, entry); err != nil {
		return canopyerr.Wrap(err, op)
	}

	return nil
}

// AppendDomain records a business-meaningful audit entry.
func (s *Service) AppendDomain(
	ctx context.Context,
	actorID string,
	action domain.Action,
	resourceType, resourceID string,
	orgID, workspaceID string,
	metadata map[string]any,
) error {
	const op = "ledger: append domain"

	entry, err := domain.NewDomainEntry(actorID, action, resourceType, resourceID, orgID, workspaceID, metadata)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := s.repo.AppendDomain(ctx, entry); err != nil {
		return canopyerr.Wrap(err, op)
	}

	return nil
}

// QuerySystem returns system entries matching the given filter.
func (s *Service) QuerySystem(ctx context.Context, filter domain.SystemFilter) ([]domain.SystemEntry, error) {
	const op = "ledger: query system"
	entries, err := s.repo.FindSystemEntries(ctx, filter)
	if err != nil {
		return nil, canopyerr.Wrap(err, op)
	}
	return entries, nil
}

// QueryDomain returns domain entries matching the given filter.
func (s *Service) QueryDomain(ctx context.Context, filter domain.DomainFilter) ([]domain.DomainEntry, error) {
	const op = "ledger: query domain"
	entries, err := s.repo.FindDomainEntries(ctx, filter)
	if err != nil {
		return nil, canopyerr.Wrap(err, op)
	}
	return entries, nil
}

// QueryWorkspaceActivity returns domain entries for a workspace.
// Caller must be a workspace member.
func (s *Service) QueryWorkspaceActivity(ctx context.Context, workspaceID types.WorkspaceID, limit int) ([]domain.DomainEntry, error) {
	const op = "ledger: query workspace activity"

	if _, err := s.requireMember(ctx, workspaceID); err != nil {
		return nil, canopyerr.Wrap(err, op)
	}

	wsID := workspaceID.String()
	filter := domain.DomainFilter{
		WorkspaceID: &wsID,
		Limit:       limit,
	}
	entries, err := s.repo.FindDomainEntries(ctx, filter)
	if err != nil {
		return nil, canopyerr.Wrap(err, op)
	}
	return entries, nil
}

// --- Auth Helpers ---

func (s *Service) requireMember(ctx context.Context, workspaceID types.WorkspaceID) (types.UserID, error) {
	claims, ok := auth.FromClaims(ctx)
	if !ok {
		return types.UserID{}, canopyerr.ErrUnauthenticated
	}
	callerID := types.UserIDFrom(claims.Subject)

	if _, err := s.wsMembers.FindMember(ctx, workspaceID, callerID); err != nil {
		if canopyerr.GetKind(err) == canopyerr.KindNotFound {
			return types.UserID{}, canopyerr.ErrUnauthorized
		}
		return types.UserID{}, err
	}
	return callerID, nil
}
