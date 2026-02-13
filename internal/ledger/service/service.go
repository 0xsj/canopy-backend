package service

import (
	"context"

	"github.com/0xsj/canopy-backend/internal/ledger/domain"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// Service implements the ledger application logic.
// The ledger is a sink — it consumes events and records entries
// but publishes no events of its own.
type Service struct {
	repo domain.LedgerRepository
	log  logger.Logger
}

// New creates a new ledger service.
func New(repo domain.LedgerRepository, log logger.Logger) *Service {
	return &Service{repo: repo, log: log}
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
