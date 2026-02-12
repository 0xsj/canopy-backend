package domain

import (
	"context"
	"time"
)

// LedgerRepository defines the persistence port for ledger entries.
// All writes are append-only — there are no update or delete operations.
type LedgerRepository interface {
	// AppendSystem persists a new system entry.
	AppendSystem(ctx context.Context, entry SystemEntry) error

	// AppendDomain persists a new domain entry.
	AppendDomain(ctx context.Context, entry DomainEntry) error

	// FindSystemEntries queries system entries with optional filters.
	FindSystemEntries(ctx context.Context, filter SystemFilter) ([]SystemEntry, error)

	// FindDomainEntries queries domain entries with optional filters.
	FindDomainEntries(ctx context.Context, filter DomainFilter) ([]DomainEntry, error)
}

// SystemFilter provides optional filtering for system entry queries.
type SystemFilter struct {
	SourceContext *string
	EventSubject  *string
	After         *time.Time
	Before        *time.Time
	Limit         int
}

// DomainFilter provides optional filtering for domain entry queries.
type DomainFilter struct {
	ActorID      *string
	Action       *Action
	ResourceType *string
	ResourceID   *string
	OrgID        *string
	WorkspaceID  *string
	After        *time.Time
	Before       *time.Time
	Limit        int
}
