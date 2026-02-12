package domain

import (
	"fmt"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// SystemEntry captures raw event data flowing through the system.
// Every domain event published to NATS is recorded as a system entry.
// System entries serve operational debugging and infrastructure-level
// auditability. They are append-only and never modified.
type SystemEntry struct {
	id            types.ID[systemEntryTag]
	eventSubject  string
	eventData     map[string]any
	sourceContext string // which bounded context published the event
	createdAt     types.Timestamp
}

type systemEntryTag struct{}

const prefixSystemEntry = "sye"

// NewSystemEntry creates a new system-level audit entry.
func NewSystemEntry(eventSubject string, eventData map[string]any, sourceContext string) (SystemEntry, error) {
	if eventSubject == "" {
		return SystemEntry{}, fmt.Errorf("ledger: event subject is required")
	}
	if sourceContext == "" {
		return SystemEntry{}, fmt.Errorf("ledger: source context is required")
	}

	if eventData == nil {
		eventData = make(map[string]any)
	}

	return SystemEntry{
		id:            types.NewID[systemEntryTag](prefixSystemEntry),
		eventSubject:  eventSubject,
		eventData:     eventData,
		sourceContext: sourceContext,
		createdAt:     types.Now(),
	}, nil
}

// ReconstructSystemEntry builds a SystemEntry from trusted data.
func ReconstructSystemEntry(
	id types.ID[systemEntryTag],
	eventSubject string,
	eventData map[string]any,
	sourceContext string,
	createdAt types.Timestamp,
) SystemEntry {
	if eventData == nil {
		eventData = make(map[string]any)
	}
	return SystemEntry{
		id:            id,
		eventSubject:  eventSubject,
		eventData:     eventData,
		sourceContext: sourceContext,
		createdAt:     createdAt,
	}
}

func (e SystemEntry) ID() types.ID[systemEntryTag] { return e.id }
func (e SystemEntry) EventSubject() string         { return e.eventSubject }
func (e SystemEntry) EventData() map[string]any    { return e.eventData }
func (e SystemEntry) SourceContext() string        { return e.sourceContext }
func (e SystemEntry) CreatedAt() types.Timestamp   { return e.createdAt }

// Action describes a domain-meaningful operation for audit purposes.
type Action string

// Common actions across contexts.
const (
	ActionCreated  Action = "created"
	ActionUpdated  Action = "updated"
	ActionDeleted  Action = "deleted"
	ActionJoined   Action = "joined"
	ActionLeft     Action = "left"
	ActionPromoted Action = "promoted"
	ActionSignaled Action = "signaled"
)

// DomainEntry captures a business-meaningful action with semantic context.
// Domain entries answer "who did what, to which resource, where" and serve
// admin dashboards, workspace activity feeds, and compliance reporting.
// They are append-only and never modified.
type DomainEntry struct {
	id           types.ID[domainEntryTag]
	actorID      string // user ID as string (primitives at boundary)
	action       Action
	resourceType string // e.g., "leaf", "workspace", "member"
	resourceID   string // the affected resource's ID
	orgID        string // optional — empty for system-level actions
	workspaceID  string // optional — empty for org-level actions
	metadata     map[string]any
	createdAt    types.Timestamp
}

type domainEntryTag struct{}

const prefixDomainEntry = "dme"

// NewDomainEntry creates a new domain-level audit entry.
func NewDomainEntry(
	actorID string,
	action Action,
	resourceType string,
	resourceID string,
	orgID string,
	workspaceID string,
	metadata map[string]any,
) (DomainEntry, error) {
	if actorID == "" {
		return DomainEntry{}, fmt.Errorf("ledger: actor ID is required")
	}
	if action == "" {
		return DomainEntry{}, fmt.Errorf("ledger: action is required")
	}
	if resourceType == "" {
		return DomainEntry{}, fmt.Errorf("ledger: resource type is required")
	}
	if resourceID == "" {
		return DomainEntry{}, fmt.Errorf("ledger: resource ID is required")
	}

	if metadata == nil {
		metadata = make(map[string]any)
	}

	return DomainEntry{
		id:           types.NewID[domainEntryTag](prefixDomainEntry),
		actorID:      actorID,
		action:       action,
		resourceType: resourceType,
		resourceID:   resourceID,
		orgID:        orgID,
		workspaceID:  workspaceID,
		metadata:     metadata,
		createdAt:    types.Now(),
	}, nil
}

// ReconstructDomainEntry builds a DomainEntry from trusted data.
func ReconstructDomainEntry(
	id types.ID[domainEntryTag],
	actorID string,
	action Action,
	resourceType string,
	resourceID string,
	orgID string,
	workspaceID string,
	metadata map[string]any,
	createdAt types.Timestamp,
) DomainEntry {
	if metadata == nil {
		metadata = make(map[string]any)
	}
	return DomainEntry{
		id:           id,
		actorID:      actorID,
		action:       action,
		resourceType: resourceType,
		resourceID:   resourceID,
		orgID:        orgID,
		workspaceID:  workspaceID,
		metadata:     metadata,
		createdAt:    createdAt,
	}
}

func (e DomainEntry) ID() types.ID[domainEntryTag] { return e.id }
func (e DomainEntry) ActorID() string              { return e.actorID }
func (e DomainEntry) Action() Action               { return e.action }
func (e DomainEntry) ResourceType() string         { return e.resourceType }
func (e DomainEntry) ResourceID() string           { return e.resourceID }
func (e DomainEntry) OrgID() string                { return e.orgID }
func (e DomainEntry) WorkspaceID() string          { return e.workspaceID }
func (e DomainEntry) Metadata() map[string]any     { return e.metadata }
func (e DomainEntry) CreatedAt() types.Timestamp   { return e.createdAt }
