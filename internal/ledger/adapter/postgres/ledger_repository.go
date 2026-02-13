package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/0xsj/canopy-backend/internal/ledger/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// LedgerRepository implements domain.LedgerRepository using Postgres.
type LedgerRepository struct {
	db database.DBTX
}

// NewLedgerRepository creates a new LedgerRepository.
func NewLedgerRepository(db database.DBTX) *LedgerRepository {
	return &LedgerRepository{db: db}
}

var _ domain.LedgerRepository = (*LedgerRepository)(nil)

func (r *LedgerRepository) AppendSystem(ctx context.Context, entry domain.SystemEntry) error {
	const op = "ledger: append system entry"
	const query = `
		INSERT INTO system_entries (id, event_subject, event_data, source_context, created_at)
		VALUES ($1, $2, $3, $4, $5)`

	eventDataJSON, err := json.Marshal(entry.EventData())
	if err != nil {
		return database.MapQueryError(err, op)
	}

	_, err = r.db.Exec(ctx, query,
		entry.ID().String(),
		entry.EventSubject(),
		eventDataJSON,
		entry.SourceContext(),
		entry.CreatedAt().Time(),
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return nil
}

func (r *LedgerRepository) AppendDomain(ctx context.Context, entry domain.DomainEntry) error {
	const op = "ledger: append domain entry"
	const query = `
		INSERT INTO domain_entries (id, actor_id, action, resource_type, resource_id, org_id, workspace_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	metadataJSON, err := json.Marshal(entry.Metadata())
	if err != nil {
		return database.MapQueryError(err, op)
	}

	_, err = r.db.Exec(ctx, query,
		entry.ID().String(),
		entry.ActorID(),
		string(entry.Action()),
		entry.ResourceType(),
		entry.ResourceID(),
		nullableString(entry.OrgID()),
		nullableString(entry.WorkspaceID()),
		metadataJSON,
		entry.CreatedAt().Time(),
	)
	if err != nil {
		return database.MapQueryError(err, op)
	}
	return nil
}

func (r *LedgerRepository) FindSystemEntries(ctx context.Context, filter domain.SystemFilter) ([]domain.SystemEntry, error) {
	const op = "ledger: find system entries"

	where := []string{"1=1"}
	args := []any{}
	idx := 1

	if filter.SourceContext != nil {
		where = append(where, fmt.Sprintf("source_context = $%d", idx))
		args = append(args, *filter.SourceContext)
		idx++
	}
	if filter.EventSubject != nil {
		where = append(where, fmt.Sprintf("event_subject = $%d", idx))
		args = append(args, *filter.EventSubject)
		idx++
	}
	if filter.After != nil {
		where = append(where, fmt.Sprintf("created_at > $%d", idx))
		args = append(args, *filter.After)
		idx++
	}
	if filter.Before != nil {
		where = append(where, fmt.Sprintf("created_at < $%d", idx))
		args = append(args, *filter.Before)
		idx++
	}

	limit := 100
	if filter.Limit > 0 {
		limit = filter.Limit
	}

	query := fmt.Sprintf(
		`SELECT id, event_subject, event_data, source_context, created_at
		FROM system_entries WHERE %s ORDER BY created_at DESC LIMIT %d`,
		strings.Join(where, " AND "), limit,
	)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	defer rows.Close()

	var entries []domain.SystemEntry
	for rows.Next() {
		e, err := scanSystemEntry(rows, op)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *LedgerRepository) FindDomainEntries(ctx context.Context, filter domain.DomainFilter) ([]domain.DomainEntry, error) {
	const op = "ledger: find domain entries"

	where := []string{"1=1"}
	args := []any{}
	idx := 1

	if filter.ActorID != nil {
		where = append(where, fmt.Sprintf("actor_id = $%d", idx))
		args = append(args, *filter.ActorID)
		idx++
	}
	if filter.Action != nil {
		where = append(where, fmt.Sprintf("action = $%d", idx))
		args = append(args, string(*filter.Action))
		idx++
	}
	if filter.ResourceType != nil {
		where = append(where, fmt.Sprintf("resource_type = $%d", idx))
		args = append(args, *filter.ResourceType)
		idx++
	}
	if filter.ResourceID != nil {
		where = append(where, fmt.Sprintf("resource_id = $%d", idx))
		args = append(args, *filter.ResourceID)
		idx++
	}
	if filter.OrgID != nil {
		where = append(where, fmt.Sprintf("org_id = $%d", idx))
		args = append(args, *filter.OrgID)
		idx++
	}
	if filter.WorkspaceID != nil {
		where = append(where, fmt.Sprintf("workspace_id = $%d", idx))
		args = append(args, *filter.WorkspaceID)
		idx++
	}
	if filter.After != nil {
		where = append(where, fmt.Sprintf("created_at > $%d", idx))
		args = append(args, *filter.After)
		idx++
	}
	if filter.Before != nil {
		where = append(where, fmt.Sprintf("created_at < $%d", idx))
		args = append(args, *filter.Before)
		idx++
	}

	limit := 100
	if filter.Limit > 0 {
		limit = filter.Limit
	}

	query := fmt.Sprintf(
		`SELECT id, actor_id, action, resource_type, resource_id, org_id, workspace_id, metadata, created_at
		FROM domain_entries WHERE %s ORDER BY created_at DESC LIMIT %d`,
		strings.Join(where, " AND "), limit,
	)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	defer rows.Close()

	var entries []domain.DomainEntry
	for rows.Next() {
		e, err := scanDomainEntry(rows, op)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// --- helpers ---

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSystemEntry(row rowScanner, op string) (domain.SystemEntry, error) {
	var (
		rawID         string
		eventSubject  string
		eventDataJSON []byte
		sourceContext string
		createdAt     time.Time
	)

	if err := row.Scan(&rawID, &eventSubject, &eventDataJSON, &sourceContext, &createdAt); err != nil {
		return domain.SystemEntry{}, database.MapQueryError(err, op)
	}

	eventData := make(map[string]any)
	if len(eventDataJSON) > 0 {
		if err := json.Unmarshal(eventDataJSON, &eventData); err != nil {
			return domain.SystemEntry{}, database.MapQueryError(err, op)
		}
	}

	return domain.ReconstructSystemEntry(
		domain.SystemEntryIDFrom(rawID),
		eventSubject,
		eventData,
		sourceContext,
		types.TimestampFrom(createdAt),
	), nil
}

func scanDomainEntry(row rowScanner, op string) (domain.DomainEntry, error) {
	var (
		rawID        string
		actorID      string
		action       string
		resourceType string
		resourceID   string
		orgID        *string
		workspaceID  *string
		metadataJSON []byte
		createdAt    time.Time
	)

	if err := row.Scan(&rawID, &actorID, &action, &resourceType, &resourceID, &orgID, &workspaceID, &metadataJSON, &createdAt); err != nil {
		return domain.DomainEntry{}, database.MapQueryError(err, op)
	}

	metadata := make(map[string]any)
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
			return domain.DomainEntry{}, database.MapQueryError(err, op)
		}
	}

	return domain.ReconstructDomainEntry(
		domain.DomainEntryIDFrom(rawID),
		actorID,
		domain.Action(action),
		resourceType,
		resourceID,
		derefString(orgID),
		derefString(workspaceID),
		metadata,
		types.TimestampFrom(createdAt),
	), nil
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
