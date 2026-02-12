package domain

import (
	"testing"

	"github.com/0xsj/canopy-backend/pkg/types"
)

func TestNewDeliverable_ValidInput(t *testing.T) {
	d, err := NewDeliverable(
		types.NewWorkspaceID(),
		FormatMarkdown,
		"# Draft\nFirst version",
		[]types.LeafID{types.NewLeafID()},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if d.Version() != 1 {
		t.Errorf("version = %d, want 1", d.Version())
	}
}

func TestNewDeliverable_RejectsEmptyContent(t *testing.T) {
	_, err := NewDeliverable(types.NewWorkspaceID(), FormatMarkdown, "", []types.LeafID{types.NewLeafID()})
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestNewDeliverable_RejectsNoSources(t *testing.T) {
	_, err := NewDeliverable(types.NewWorkspaceID(), FormatMarkdown, "content", nil)
	if err == nil {
		t.Fatal("expected error for no source leaves")
	}
}

func TestUpdateContent_IncrementsVersion(t *testing.T) {
	d, _ := NewDeliverable(types.NewWorkspaceID(), FormatMarkdown, "v1", []types.LeafID{types.NewLeafID()})

	if err := d.UpdateContent("v2"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Version() != 2 {
		t.Errorf("version = %d, want 2", d.Version())
	}
	if d.Content() != "v2" {
		t.Errorf("content = %q, want %q", d.Content(), "v2")
	}
}

func TestUpdateContent_RejectsSameContent(t *testing.T) {
	d, _ := NewDeliverable(types.NewWorkspaceID(), FormatMarkdown, "same", []types.LeafID{types.NewLeafID()})

	err := d.UpdateContent("same")
	if err == nil {
		t.Fatal("expected error for same content")
	}
}

func TestChangeFormat(t *testing.T) {
	d, _ := NewDeliverable(types.NewWorkspaceID(), FormatMarkdown, "content", []types.LeafID{types.NewLeafID()})

	if err := d.ChangeFormat(FormatPDF); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Format() != FormatPDF {
		t.Errorf("format = %q, want %q", d.Format(), FormatPDF)
	}
}

func TestAddSourceLeaf_Deduplicates(t *testing.T) {
	leafID := types.NewLeafID()
	d, _ := NewDeliverable(types.NewWorkspaceID(), FormatMarkdown, "content", []types.LeafID{leafID})

	err := d.AddSourceLeaf(leafID)
	if err == nil {
		t.Fatal("expected error for duplicate source leaf")
	}

	newLeaf := types.NewLeafID()
	if err := d.AddSourceLeaf(newLeaf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(d.SourceLeafIDs()) != 2 {
		t.Errorf("source count = %d, want 2", len(d.SourceLeafIDs()))
	}
}
