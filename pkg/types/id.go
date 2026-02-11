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
func NewWorkspaceID() WorkspaceID     { return NewID[workspaceTag](PrefixWorkspace) }
func NewSeedID() SeedID               { return NewID[seedTag](PrefixSeed) }
func NewLeafID() LeafID               { return NewID[leafTag](PrefixLeaf) }
func NewBranchID() BranchID           { return NewID[branchTag](PrefixBranch) }
func NewConnectionID() ConnectionID   { return NewID[connectionTag](PrefixConnection) }
func NewThreadID() ThreadID           { return NewID[threadTag](PrefixThread) }
func NewCheckpointID() CheckpointID   { return NewID[checkpointTag](PrefixCheckpoint) }
func NewDeliverableID() DeliverableID { return NewID[deliverableTag](PrefixDeliverable) }
