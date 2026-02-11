package types

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewID_HasPrefix(t *testing.T) {
	id := NewUserID()
	if !strings.HasPrefix(id.String(), PrefixUser+"_") {
		t.Errorf("UserID = %q, want prefix %q", id.String(), PrefixUser+"_")
	}
}

func TestNewID_UniqueEachCall(t *testing.T) {
	a := NewLeafID()
	b := NewLeafID()
	if a.String() == b.String() {
		t.Errorf("two generated IDs should not be equal: %q", a.String())
	}
}

func TestNewID_AllPrefixes(t *testing.T) {
	tests := []struct {
		name   string
		gen    func() string
		prefix string
	}{
		{"User", func() string { return NewUserID().String() }, PrefixUser},
		{"Workspace", func() string { return NewWorkspaceID().String() }, PrefixWorkspace},
		{"Seed", func() string { return NewSeedID().String() }, PrefixSeed},
		{"Leaf", func() string { return NewLeafID().String() }, PrefixLeaf},
		{"Branch", func() string { return NewBranchID().String() }, PrefixBranch},
		{"Connection", func() string { return NewConnectionID().String() }, PrefixConnection},
		{"Thread", func() string { return NewThreadID().String() }, PrefixThread},
		{"Checkpoint", func() string { return NewCheckpointID().String() }, PrefixCheckpoint},
		{"Deliverable", func() string { return NewDeliverableID().String() }, PrefixDeliverable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := tt.gen()
			if !strings.HasPrefix(id, tt.prefix+"_") {
				t.Errorf("%s ID = %q, want prefix %q", tt.name, id, tt.prefix+"_")
			}
		})
	}
}

func TestParseID_Valid(t *testing.T) {
	original := NewLeafID()
	parsed, err := ParseID[leafTag](original.String(), PrefixLeaf)
	if err != nil {
		t.Fatalf("ParseID failed: %v", err)
	}
	if parsed.String() != original.String() {
		t.Errorf("parsed = %q, want %q", parsed.String(), original.String())
	}
}

func TestParseID_WrongPrefix(t *testing.T) {
	id := NewLeafID()
	_, err := ParseID[userTag](id.String(), PrefixUser)
	if err == nil {
		t.Error("ParseID should fail with wrong prefix")
	}
}

func TestParseID_Empty(t *testing.T) {
	_, err := ParseID[leafTag]("", PrefixLeaf)
	if err == nil {
		t.Error("ParseID should fail on empty string")
	}
}

func TestParseID_NoUnderscore(t *testing.T) {
	_, err := ParseID[leafTag]("nounderscore", PrefixLeaf)
	if err == nil {
		t.Error("ParseID should fail without underscore separator")
	}
}

func TestIDFrom_TrustedInput(t *testing.T) {
	raw := "leaf_abc123"
	id := IDFrom[leafTag](raw)
	if id.String() != raw {
		t.Errorf("IDFrom = %q, want %q", id.String(), raw)
	}
}

func TestID_IsZero(t *testing.T) {
	var id LeafID
	if !id.IsZero() {
		t.Error("zero-value ID should be zero")
	}

	generated := NewLeafID()
	if generated.IsZero() {
		t.Error("generated ID should not be zero")
	}
}

func TestID_JSONRoundTrip(t *testing.T) {
	original := NewLeafID()

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var parsed LeafID
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if parsed.String() != original.String() {
		t.Errorf("round-trip: got %q, want %q", parsed.String(), original.String())
	}
}

func TestID_JSONMarshalFormat(t *testing.T) {
	id := IDFrom[leafTag]("leaf_abc123")
	data, _ := json.Marshal(id)

	want := `"leaf_abc123"`
	if string(data) != want {
		t.Errorf("JSON = %s, want %s", data, want)
	}
}
