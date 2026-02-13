package postgres

import (
	"context"

	"github.com/0xsj/canopy-backend/internal/ledger/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/ledger/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
)

// LedgerRepository implements domain.LedgerRepository using Postgres via sqlc.
type LedgerRepository struct {
	q *sqlc.Queries
}

// NewLedgerRepository creates a new LedgerRepository.
func NewLedgerRepository(db database.DBTX) *LedgerRepository {
	return &LedgerRepository{q: sqlc.New(db)}
}

var _ domain.LedgerRepository = (*LedgerRepository)(nil)

func (r *LedgerRepository) AppendSystem(ctx context.Context, entry domain.SystemEntry) error {
	const op = "ledger: append system entry"
	params, err := systemEntryToAppendParams(entry)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return database.MapQueryError(r.q.AppendSystemEntry(ctx, params), op)
}

func (r *LedgerRepository) AppendDomain(ctx context.Context, entry domain.DomainEntry) error {
	const op = "ledger: append domain entry"
	params, err := domainEntryToAppendParams(entry)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return database.MapQueryError(r.q.AppendDomainEntry(ctx, params), op)
}

func (r *LedgerRepository) FindSystemEntries(ctx context.Context, filter domain.SystemFilter) ([]domain.SystemEntry, error) {
	const op = "ledger: find system entries"
	rows, err := r.q.FindSystemEntries(ctx, systemFilterToParams(filter))
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	entries, err := systemEntriesToDomain(rows)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return entries, nil
}

func (r *LedgerRepository) FindDomainEntries(ctx context.Context, filter domain.DomainFilter) ([]domain.DomainEntry, error) {
	const op = "ledger: find domain entries"
	rows, err := r.q.FindDomainEntries(ctx, domainFilterToParams(filter))
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	entries, err := domainEntriesToDomain(rows)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return entries, nil
}
