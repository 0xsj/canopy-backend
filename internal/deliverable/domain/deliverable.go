package domain

import (
	"fmt"
	"strings"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// Format represents the output format of a deliverable.
type Format string

const (
	FormatMarkdown Format = "markdown"
	FormatPDF      Format = "pdf"
	FormatHTML     Format = "html"
)

// IsValid reports whether the format is a recognized value.
func (f Format) IsValid() bool {
	switch f {
	case FormatMarkdown, FormatPDF, FormatHTML:
		return true
	}
	return false
}

// Deliverable is the emergent output artifact produced during the emergent
// phase of a workspace. This is the one entity in Canopy where mutability
// is a feature — deliverables are drafted, edited, and versioned.
type Deliverable struct {
	id            types.DeliverableID
	workspaceID   types.WorkspaceID
	format        Format
	content       string
	sourceLeafIDs []types.LeafID
	version       int
	timestamps    types.Timestamps
}

// NewDeliverable creates a new deliverable draft from consensus-backed leaves.
func NewDeliverable(
	workspaceID types.WorkspaceID,
	format Format,
	content string,
	sourceLeafIDs []types.LeafID,
) (Deliverable, error) {
	if workspaceID.IsZero() {
		return Deliverable{}, fmt.Errorf("deliverable: workspace ID is required")
	}
	if !format.IsValid() {
		return Deliverable{}, fmt.Errorf("deliverable: invalid format %q", format)
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return Deliverable{}, fmt.Errorf("deliverable: content is required")
	}

	if len(sourceLeafIDs) == 0 {
		return Deliverable{}, fmt.Errorf("deliverable: at least one source leaf is required")
	}

	return Deliverable{
		id:            types.NewDeliverableID(),
		workspaceID:   workspaceID,
		format:        format,
		content:       content,
		sourceLeafIDs: sourceLeafIDs,
		version:       1,
		timestamps:    types.NewMutableTimestamps(),
	}, nil
}

// ReconstructDeliverable builds a Deliverable from trusted data.
func ReconstructDeliverable(
	id types.DeliverableID,
	workspaceID types.WorkspaceID,
	format Format,
	content string,
	sourceLeafIDs []types.LeafID,
	version int,
	timestamps types.Timestamps,
) Deliverable {
	return Deliverable{
		id:            id,
		workspaceID:   workspaceID,
		format:        format,
		content:       content,
		sourceLeafIDs: sourceLeafIDs,
		version:       version,
		timestamps:    timestamps,
	}
}

// UpdateContent replaces the deliverable's content and increments the version.
func (d *Deliverable) UpdateContent(content string) error {
	content = strings.TrimSpace(content)
	if content == "" {
		return fmt.Errorf("deliverable: content is required")
	}
	if content == d.content {
		return fmt.Errorf("deliverable: no changes")
	}

	d.content = content
	d.version++
	d.timestamps.Touch()
	return nil
}

// ChangeFormat updates the deliverable's output format.
func (d *Deliverable) ChangeFormat(format Format) error {
	if !format.IsValid() {
		return fmt.Errorf("deliverable: invalid format %q", format)
	}
	if format == d.format {
		return fmt.Errorf("deliverable: already in %s format", format)
	}

	d.format = format
	d.timestamps.Touch()
	return nil
}

// AddSourceLeaf adds a leaf to the deliverable's source attribution.
func (d *Deliverable) AddSourceLeaf(leafID types.LeafID) error {
	if leafID.IsZero() {
		return fmt.Errorf("deliverable: leaf ID is required")
	}
	for _, existing := range d.sourceLeafIDs {
		if existing == leafID {
			return fmt.Errorf("deliverable: leaf already in sources")
		}
	}
	d.sourceLeafIDs = append(d.sourceLeafIDs, leafID)
	d.timestamps.Touch()
	return nil
}

func (d Deliverable) ID() types.DeliverableID        { return d.id }
func (d Deliverable) WorkspaceID() types.WorkspaceID { return d.workspaceID }
func (d Deliverable) Format() Format                 { return d.format }
func (d Deliverable) Content() string                { return d.content }
func (d Deliverable) SourceLeafIDs() []types.LeafID  { return d.sourceLeafIDs }
func (d Deliverable) Version() int                   { return d.version }
func (d Deliverable) Timestamps() types.Timestamps   { return d.timestamps }
