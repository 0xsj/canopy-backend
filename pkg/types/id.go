package types

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
)

// ID is a generic typed identifier. The prefix provides human-readable
// context (e.g., "usr", "ws", "leaf") and the value is a random hex string.
// Format: "{prefix}_{hex}" — e.g., "usr_a1b2c3d4e5f6".
type ID[T any] struct {
	value string
}

const idByteLength = 12 // 12 bytes = 24 hex chars

// NewID generates a new random ID with the given prefix.
func NewID[T any](prefix string) ID[T] {
	b := make([]byte, idByteLength)
	if _, err := rand.Read(b); err != nil {
		panic("types: failed to generate random ID: " + err.Error())
	}
	return ID[T]{value: prefix + "_" + fmt.Sprintf("%x", b)}
}

// ParseID parses a string into a typed ID, validating the expected prefix.
func ParseID[T any](raw string, expectedPrefix string) (ID[T], error) {
	if raw == "" {
		return ID[T]{}, fmt.Errorf("types: empty ID")
	}
	prefix, _, found := strings.Cut(raw, "_")
	if !found {
		return ID[T]{}, fmt.Errorf("types: invalid ID format: %q", raw)
	}
	if prefix != expectedPrefix {
		return ID[T]{}, fmt.Errorf("types: expected prefix %q, got %q in %q", expectedPrefix, prefix, raw)
	}
	return ID[T]{value: raw}, nil
}

// IDFrom creates an ID from a trusted string value (e.g., from the database).
// No validation is performed — use ParseID for untrusted input.
func IDFrom[T any](raw string) ID[T] {
	return ID[T]{value: raw}
}

// String returns the full ID string.
func (id ID[T]) String() string {
	return id.value
}

// IsZero reports whether the ID is empty/unset.
func (id ID[T]) IsZero() bool {
	return id.value == ""
}

// MarshalJSON implements json.Marshaler.
func (id ID[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.value)
}

// UnmarshalJSON implements json.Unmarshaler.
func (id *ID[T]) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &id.value)
}

// Concrete ID types — one per entity that crosses context boundaries.
// The phantom type parameter prevents mixing IDs across entity types.

type (
	userTag        struct{}
	orgTag         struct{}
	teamTag        struct{}
	workspaceTag   struct{}
	seedTag        struct{}
	leafTag        struct{}
	branchTag      struct{}
	connectionTag  struct{}
	threadTag      struct{}
	checkpointTag  struct{}
	deliverableTag struct{}
)

type (
	UserID        = ID[userTag]
	OrgID         = ID[orgTag]
	TeamID        = ID[teamTag]
	WorkspaceID   = ID[workspaceTag]
	SeedID        = ID[seedTag]
	LeafID        = ID[leafTag]
	BranchID      = ID[branchTag]
	ConnectionID  = ID[connectionTag]
	ThreadID      = ID[threadTag]
	CheckpointID  = ID[checkpointTag]
	DeliverableID = ID[deliverableTag]
)

// ID prefixes — used by generators and parsers.
const (
	PrefixUser        = "usr"
	PrefixOrg         = "org"
	PrefixTeam        = "team"
	PrefixWorkspace   = "ws"
	PrefixSeed        = "seed"
	PrefixLeaf        = "leaf"
	PrefixBranch      = "br"
	PrefixConnection  = "conn"
	PrefixThread      = "thr"
	PrefixCheckpoint  = "chk"
	PrefixDeliverable = "del"
)

// Convenience generators for each entity type.

func NewUserID() UserID               { return NewID[userTag](PrefixUser) }
func NewOrgID() OrgID                 { return NewID[orgTag](PrefixOrg) }
func NewTeamID() TeamID               { return NewID[teamTag](PrefixTeam) }
func NewWorkspaceID() WorkspaceID     { return NewID[workspaceTag](PrefixWorkspace) }
func NewSeedID() SeedID               { return NewID[seedTag](PrefixSeed) }
func NewLeafID() LeafID               { return NewID[leafTag](PrefixLeaf) }
func NewBranchID() BranchID           { return NewID[branchTag](PrefixBranch) }
func NewConnectionID() ConnectionID   { return NewID[connectionTag](PrefixConnection) }
func NewThreadID() ThreadID           { return NewID[threadTag](PrefixThread) }
func NewCheckpointID() CheckpointID   { return NewID[checkpointTag](PrefixCheckpoint) }
func NewDeliverableID() DeliverableID { return NewID[deliverableTag](PrefixDeliverable) }

// Reconstruction helpers — create typed IDs from trusted database strings.
// No validation is performed; use ParseID for untrusted input.

func UserIDFrom(raw string) UserID               { return IDFrom[userTag](raw) }
func OrgIDFrom(raw string) OrgID                 { return IDFrom[orgTag](raw) }
func TeamIDFrom(raw string) TeamID               { return IDFrom[teamTag](raw) }
func WorkspaceIDFrom(raw string) WorkspaceID     { return IDFrom[workspaceTag](raw) }
func SeedIDFrom(raw string) SeedID               { return IDFrom[seedTag](raw) }
func LeafIDFrom(raw string) LeafID               { return IDFrom[leafTag](raw) }
func BranchIDFrom(raw string) BranchID           { return IDFrom[branchTag](raw) }
func ConnectionIDFrom(raw string) ConnectionID   { return IDFrom[connectionTag](raw) }
func ThreadIDFrom(raw string) ThreadID           { return IDFrom[threadTag](raw) }
func CheckpointIDFrom(raw string) CheckpointID   { return IDFrom[checkpointTag](raw) }
func DeliverableIDFrom(raw string) DeliverableID { return IDFrom[deliverableTag](raw) }

// Parse helpers — validate untrusted input (e.g., path params, request bodies).
// These wrap ParseID with the correct tag and prefix so callers outside this
// package don't need access to the unexported tag types.

func ParseUserID(raw string) (UserID, error) { return ParseID[userTag](raw, PrefixUser) }
func ParseOrgID(raw string) (OrgID, error)   { return ParseID[orgTag](raw, PrefixOrg) }
func ParseTeamID(raw string) (TeamID, error) { return ParseID[teamTag](raw, PrefixTeam) }
func ParseWorkspaceID(raw string) (WorkspaceID, error) {
	return ParseID[workspaceTag](raw, PrefixWorkspace)
}
func ParseSeedID(raw string) (SeedID, error)     { return ParseID[seedTag](raw, PrefixSeed) }
func ParseLeafID(raw string) (LeafID, error)     { return ParseID[leafTag](raw, PrefixLeaf) }
func ParseBranchID(raw string) (BranchID, error) { return ParseID[branchTag](raw, PrefixBranch) }
func ParseConnectionID(raw string) (ConnectionID, error) {
	return ParseID[connectionTag](raw, PrefixConnection)
}
func ParseThreadID(raw string) (ThreadID, error) { return ParseID[threadTag](raw, PrefixThread) }
func ParseCheckpointID(raw string) (CheckpointID, error) {
	return ParseID[checkpointTag](raw, PrefixCheckpoint)
}
func ParseDeliverableID(raw string) (DeliverableID, error) {
	return ParseID[deliverableTag](raw, PrefixDeliverable)
}
