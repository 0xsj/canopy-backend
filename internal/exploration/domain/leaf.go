package domain

import (
	"fmt"
	"strings"

	"github.com/0xsj/canopy-backend/pkg/types"
)

// Layer represents a leaf's position in the workspace canopy.
type Layer string

const (
	LayerUnderstory Layer = "understory"
	LayerCanopy     Layer = "canopy"
)

// IsValid reports whether the layer is a recognized value.
func (l Layer) IsValid() bool {
	switch l {
	case LayerUnderstory, LayerCanopy:
		return true
	}
	return false
}

// Source records lineage for synthesis leaves — which leaf contributed
// to this one.
type Source struct {
	LeafID string `json:"leaf_id"`
	Title  string `json:"title"`
}

// Leaf is the core unit of thought in Canopy. A leaf captures a structured
// insight produced through AI-assisted exploration. Once confirmed, a leaf
// is immutable — no update methods exist, and there is no updated_at column.
type Leaf struct {
	id            types.LeafID
	workspaceID   types.WorkspaceID
	seedID        types.SeedID
	branchID      types.BranchID
	authorID      types.UserID
	parentLeafID  types.LeafID // zero if root leaf of a branch
	title         string
	summary       string
	keyPoints     []string
	openQuestions []string
	tags          []string
	layer         Layer
	sources       []Source // non-nil only for synthesis leaves
	metadata      map[string]any
	timestamps    types.Timestamps // immutable — created_at only
}

// NewLeaf creates a new leaf in the understory layer. Leaves start in the
// understory and can only be promoted to the canopy through the convergence
// context. The leaf is immutable after this call.
func NewLeaf(
	workspaceID types.WorkspaceID,
	seedID types.SeedID,
	branchID types.BranchID,
	authorID types.UserID,
	parentLeafID types.LeafID,
	title string,
	summary string,
	keyPoints []string,
	openQuestions []string,
	tags []string,
) (Leaf, error) {
	if workspaceID.IsZero() {
		return Leaf{}, fmt.Errorf("exploration: workspace ID is required")
	}
	if seedID.IsZero() {
		return Leaf{}, fmt.Errorf("exploration: seed ID is required")
	}
	if branchID.IsZero() {
		return Leaf{}, fmt.Errorf("exploration: branch ID is required")
	}
	if authorID.IsZero() {
		return Leaf{}, fmt.Errorf("exploration: author ID is required")
	}

	title = strings.TrimSpace(title)
	if title == "" {
		return Leaf{}, fmt.Errorf("exploration: leaf title is required")
	}

	summary = strings.TrimSpace(summary)
	if summary == "" {
		return Leaf{}, fmt.Errorf("exploration: leaf summary is required")
	}

	return Leaf{
		id:            types.NewLeafID(),
		workspaceID:   workspaceID,
		seedID:        seedID,
		branchID:      branchID,
		authorID:      authorID,
		parentLeafID:  parentLeafID,
		title:         title,
		summary:       summary,
		keyPoints:     keyPoints,
		openQuestions: openQuestions,
		tags:          tags,
		layer:         LayerUnderstory,
		metadata:      make(map[string]any),
		timestamps:    types.NewTimestamps(),
	}, nil
}

// NewSynthesisLeaf creates a leaf produced by the synthesis workflow.
// Synthesis leaves carry source attribution linking back to the leaves
// that were combined to produce them.
func NewSynthesisLeaf(
	workspaceID types.WorkspaceID,
	seedID types.SeedID,
	branchID types.BranchID,
	authorID types.UserID,
	title string,
	summary string,
	keyPoints []string,
	openQuestions []string,
	tags []string,
	sources []Source,
) (Leaf, error) {
	if len(sources) < 2 {
		return Leaf{}, fmt.Errorf("exploration: synthesis requires at least 2 source leaves")
	}

	leaf, err := NewLeaf(workspaceID, seedID, branchID, authorID, types.LeafID{}, title, summary, keyPoints, openQuestions, tags)
	if err != nil {
		return Leaf{}, err
	}
	leaf.sources = sources
	return leaf, nil
}

// ReconstructLeaf builds a Leaf from trusted data.
func ReconstructLeaf(
	id types.LeafID,
	workspaceID types.WorkspaceID,
	seedID types.SeedID,
	branchID types.BranchID,
	authorID types.UserID,
	parentLeafID types.LeafID,
	title string,
	summary string,
	keyPoints []string,
	openQuestions []string,
	tags []string,
	layer Layer,
	sources []Source,
	metadata map[string]any,
	timestamps types.Timestamps,
) Leaf {
	if metadata == nil {
		metadata = make(map[string]any)
	}
	return Leaf{
		id:            id,
		workspaceID:   workspaceID,
		seedID:        seedID,
		branchID:      branchID,
		authorID:      authorID,
		parentLeafID:  parentLeafID,
		title:         title,
		summary:       summary,
		keyPoints:     keyPoints,
		openQuestions: openQuestions,
		tags:          tags,
		layer:         layer,
		sources:       sources,
		metadata:      metadata,
		timestamps:    timestamps,
	}
}

// IsSynthesis reports whether this leaf was produced by synthesis.
func (l Leaf) IsSynthesis() bool { return len(l.sources) > 0 }

// IsRoot reports whether this leaf is the root of its branch (no parent).
func (l Leaf) IsRoot() bool { return l.parentLeafID.IsZero() }

func (l Leaf) ID() types.LeafID               { return l.id }
func (l Leaf) WorkspaceID() types.WorkspaceID { return l.workspaceID }
func (l Leaf) SeedID() types.SeedID           { return l.seedID }
func (l Leaf) BranchID() types.BranchID       { return l.branchID }
func (l Leaf) AuthorID() types.UserID         { return l.authorID }
func (l Leaf) ParentLeafID() types.LeafID     { return l.parentLeafID }
func (l Leaf) Title() string                  { return l.title }
func (l Leaf) Summary() string                { return l.summary }
func (l Leaf) KeyPoints() []string            { return l.keyPoints }
func (l Leaf) OpenQuestions() []string        { return l.openQuestions }
func (l Leaf) Tags() []string                 { return l.tags }
func (l Leaf) Layer() Layer                   { return l.layer }
func (l Leaf) Sources() []Source              { return l.sources }
func (l Leaf) Metadata() map[string]any       { return l.metadata }
func (l Leaf) Timestamps() types.Timestamps   { return l.timestamps }
