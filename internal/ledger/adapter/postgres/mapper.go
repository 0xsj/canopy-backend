package postgres

import (
	"encoding/json"

	"github.com/0xsj/canopy-backend/internal/ledger/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/ledger/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// --- system entry: domain → sqlc params ---

func systemEntryToAppendParams(e domain.SystemEntry) (sqlc.AppendSystemEntryParams, error) {
	eventDataJSON, err := json.Marshal(e.EventData())
	if err != nil {
		return sqlc.AppendSystemEntryParams{}, err
	}
	return sqlc.AppendSystemEntryParams{
		ID:            e.ID().String(),
		EventSubject:  e.EventSubject(),
		EventData:     eventDataJSON,
		SourceContext: e.SourceContext(),
		CreatedAt:     e.CreatedAt().Time(),
	}, nil
}

func systemFilterToParams(f domain.SystemFilter) sqlc.FindSystemEntriesParams {
	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}
	return sqlc.FindSystemEntriesParams{
		SourceContext: f.SourceContext,
		EventSubject:  f.EventSubject,
		After:         f.After,
		Before:        f.Before,
		QueryLimit:    limit,
	}
}

// --- system entry: sqlc model → domain ---

func systemEntryToDomain(row sqlc.SystemEntry) (domain.SystemEntry, error) {
	eventData := make(map[string]any)
	if len(row.EventData) > 0 {
		if err := json.Unmarshal(row.EventData, &eventData); err != nil {
			return domain.SystemEntry{}, err
		}
	}
	return domain.ReconstructSystemEntry(
		domain.SystemEntryIDFrom(row.ID),
		row.EventSubject,
		eventData,
		row.SourceContext,
		types.TimestampFrom(row.CreatedAt),
	), nil
}

func systemEntriesToDomain(rows []sqlc.SystemEntry) ([]domain.SystemEntry, error) {
	entries := make([]domain.SystemEntry, len(rows))
	for i, row := range rows {
		e, err := systemEntryToDomain(row)
		if err != nil {
			return nil, err
		}
		entries[i] = e
	}
	return entries, nil
}

// --- domain entry: domain → sqlc params ---

func domainEntryToAppendParams(e domain.DomainEntry) (sqlc.AppendDomainEntryParams, error) {
	metadataJSON, err := json.Marshal(e.Metadata())
	if err != nil {
		return sqlc.AppendDomainEntryParams{}, err
	}
	return sqlc.AppendDomainEntryParams{
		ID:           e.ID().String(),
		ActorID:      e.ActorID(),
		Action:       string(e.Action()),
		ResourceType: e.ResourceType(),
		ResourceID:   e.ResourceID(),
		OrgID:        database.NullableString(e.OrgID()),
		WorkspaceID:  database.NullableString(e.WorkspaceID()),
		Metadata:     metadataJSON,
		CreatedAt:    e.CreatedAt().Time(),
	}, nil
}

func domainFilterToParams(f domain.DomainFilter) sqlc.FindDomainEntriesParams {
	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}
	p := sqlc.FindDomainEntriesParams{
		ActorID:      f.ActorID,
		ResourceType: f.ResourceType,
		ResourceID:   f.ResourceID,
		OrgID:        f.OrgID,
		WorkspaceID:  f.WorkspaceID,
		After:        f.After,
		Before:       f.Before,
		QueryLimit:   limit,
	}
	if f.Action != nil {
		s := string(*f.Action)
		p.Action = &s
	}
	return p
}

// --- domain entry: sqlc model → domain ---

func domainEntryToDomain(row sqlc.DomainEntry) (domain.DomainEntry, error) {
	metadata := make(map[string]any)
	if len(row.Metadata) > 0 {
		if err := json.Unmarshal(row.Metadata, &metadata); err != nil {
			return domain.DomainEntry{}, err
		}
	}
	return domain.ReconstructDomainEntry(
		domain.DomainEntryIDFrom(row.ID),
		row.ActorID,
		domain.Action(row.Action),
		row.ResourceType,
		row.ResourceID,
		database.DerefString(row.OrgID),
		database.DerefString(row.WorkspaceID),
		metadata,
		types.TimestampFrom(row.CreatedAt),
	), nil
}

func domainEntriesToDomain(rows []sqlc.DomainEntry) ([]domain.DomainEntry, error) {
	entries := make([]domain.DomainEntry, len(rows))
	for i, row := range rows {
		e, err := domainEntryToDomain(row)
		if err != nil {
			return nil, err
		}
		entries[i] = e
	}
	return entries, nil
}
