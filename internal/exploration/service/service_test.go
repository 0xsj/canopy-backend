package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/0xsj/canopy-backend/internal/exploration/domain"
	wsdomain "github.com/0xsj/canopy-backend/internal/workspace/domain"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// --- Test doubles ---

type stubLeafRepo struct{}

func (s *stubLeafRepo) Create(_ context.Context, _ domain.Leaf) error { return nil }
func (s *stubLeafRepo) FindByID(_ context.Context, _ types.LeafID) (domain.Leaf, error) {
	return domain.Leaf{}, nil
}
func (s *stubLeafRepo) FindByIDs(_ context.Context, _ []types.LeafID) ([]domain.Leaf, error) {
	return nil, nil
}
func (s *stubLeafRepo) FindByWorkspace(_ context.Context, _ types.WorkspaceID, _ domain.LeafFilter) ([]domain.Leaf, error) {
	return nil, nil
}
func (s *stubLeafRepo) FindByBranch(_ context.Context, _ types.BranchID) ([]domain.Leaf, error) {
	return nil, nil
}
func (s *stubLeafRepo) FindBySeed(_ context.Context, _ types.SeedID) ([]domain.Leaf, error) {
	return nil, nil
}
func (s *stubLeafRepo) UpdateLayer(_ context.Context, _ types.LeafID, _ domain.Layer) error {
	return nil
}
func (s *stubLeafRepo) UpdatePosition(_ context.Context, _ types.LeafID, _, _ float64) error {
	return nil
}
func (s *stubLeafRepo) Search(_ context.Context, _ types.WorkspaceID, _ string) ([]domain.Leaf, error) {
	return nil, nil
}

type stubBranchRepo struct{}

func (s *stubBranchRepo) Create(_ context.Context, _ domain.Branch) error { return nil }
func (s *stubBranchRepo) FindByID(_ context.Context, _ types.BranchID) (domain.Branch, error) {
	return domain.Branch{}, nil
}
func (s *stubBranchRepo) FindByWorkspace(_ context.Context, _ types.WorkspaceID) ([]domain.Branch, error) {
	return nil, nil
}
func (s *stubBranchRepo) FindBySeed(_ context.Context, _ types.SeedID) ([]domain.Branch, error) {
	return nil, nil
}
func (s *stubBranchRepo) FindByAuthor(_ context.Context, _ types.WorkspaceID, _ types.UserID) ([]domain.Branch, error) {
	return nil, nil
}
func (s *stubBranchRepo) Update(_ context.Context, _ domain.Branch) error { return nil }

type stubConnRepo struct{}

func (s *stubConnRepo) Create(_ context.Context, _ domain.Connection) error { return nil }
func (s *stubConnRepo) FindByID(_ context.Context, _ types.ConnectionID) (domain.Connection, error) {
	return domain.Connection{}, nil
}
func (s *stubConnRepo) FindByWorkspace(_ context.Context, _ types.WorkspaceID) ([]domain.Connection, error) {
	return nil, nil
}
func (s *stubConnRepo) FindByLeaf(_ context.Context, _ types.LeafID) ([]domain.Connection, error) {
	return nil, nil
}

type stubGraphEngine struct {
	ancestorsFn    func(ctx context.Context, leafID types.LeafID) ([]domain.Leaf, error)
	descendantsFn  func(ctx context.Context, leafID types.LeafID) ([]domain.Leaf, error)
	neighborhoodFn func(ctx context.Context, leafID types.LeafID, depth int) ([]domain.Leaf, error)
	lineageFn      func(ctx context.Context, leafID types.LeafID) ([]domain.Leaf, error)
}

func (s *stubGraphEngine) Ancestors(ctx context.Context, leafID types.LeafID) ([]domain.Leaf, error) {
	if s.ancestorsFn != nil {
		return s.ancestorsFn(ctx, leafID)
	}
	return nil, nil
}

func (s *stubGraphEngine) Descendants(ctx context.Context, leafID types.LeafID) ([]domain.Leaf, error) {
	if s.descendantsFn != nil {
		return s.descendantsFn(ctx, leafID)
	}
	return nil, nil
}

func (s *stubGraphEngine) Neighborhood(ctx context.Context, leafID types.LeafID, depth int) ([]domain.Leaf, error) {
	if s.neighborhoodFn != nil {
		return s.neighborhoodFn(ctx, leafID, depth)
	}
	return nil, nil
}

func (s *stubGraphEngine) SynthesisLineage(ctx context.Context, leafID types.LeafID) ([]domain.Leaf, error) {
	if s.lineageFn != nil {
		return s.lineageFn(ctx, leafID)
	}
	return nil, nil
}

type stubWSMembers struct{}

func (s *stubWSMembers) FindMember(_ context.Context, _ types.WorkspaceID, _ types.UserID) (wsdomain.WorkspaceMember, error) {
	return wsdomain.WorkspaceMember{}, nil
}

type stubPublisher struct {
	published []events.Event
}

func (s *stubPublisher) Publish(_ context.Context, event events.Event) error {
	s.published = append(s.published, event)
	return nil
}

// --- Interface conformance ---

var _ domain.LeafRepository = (*stubLeafRepo)(nil)
var _ domain.BranchRepository = (*stubBranchRepo)(nil)
var _ domain.ConnectionRepository = (*stubConnRepo)(nil)
var _ domain.GraphQueryEngine = (*stubGraphEngine)(nil)
var _ WorkspaceMemberReader = (*stubWSMembers)(nil)
var _ events.Publisher = (*stubPublisher)(nil)

// --- Test helpers ---

func newTestService(graph *stubGraphEngine) *Service {
	return New(
		&stubLeafRepo{}, &stubBranchRepo{}, &stubConnRepo{}, graph,
		&stubWSMembers{}, nil, nil, nil,
		&stubPublisher{}, logger.NewNoop(),
	)
}

func makeLeaf(title string) domain.Leaf {
	return domain.ReconstructLeaf(
		types.NewLeafID(),
		types.NewWorkspaceID(),
		types.NewSeedID(),
		types.NewBranchID(),
		types.NewUserID(),
		types.LeafID{},
		title, "summary",
		nil, nil, nil,
		domain.LayerUnderstory,
		nil, nil,
		0, 0,
		types.NewTimestamps(),
	)
}

// --- GetAncestors tests ---

func TestGetAncestors_ReturnsLeaves(t *testing.T) {
	parent := makeLeaf("parent")
	grandparent := makeLeaf("grandparent")

	graph := &stubGraphEngine{
		ancestorsFn: func(_ context.Context, _ types.LeafID) ([]domain.Leaf, error) {
			return []domain.Leaf{parent, grandparent}, nil
		},
	}
	svc := newTestService(graph)

	leaves, err := svc.GetAncestors(context.Background(), types.NewLeafID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(leaves) != 2 {
		t.Fatalf("got %d leaves, want 2", len(leaves))
	}
	if leaves[0].Title() != "parent" {
		t.Errorf("first ancestor title = %q, want %q", leaves[0].Title(), "parent")
	}
}

func TestGetAncestors_EmptyResult(t *testing.T) {
	graph := &stubGraphEngine{
		ancestorsFn: func(_ context.Context, _ types.LeafID) ([]domain.Leaf, error) {
			return nil, nil
		},
	}
	svc := newTestService(graph)

	leaves, err := svc.GetAncestors(context.Background(), types.NewLeafID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(leaves) != 0 {
		t.Fatalf("got %d leaves, want 0", len(leaves))
	}
}

func TestGetAncestors_WrapsError(t *testing.T) {
	graph := &stubGraphEngine{
		ancestorsFn: func(_ context.Context, _ types.LeafID) ([]domain.Leaf, error) {
			return nil, fmt.Errorf("db connection failed")
		},
	}
	svc := newTestService(graph)

	_, err := svc.GetAncestors(context.Background(), types.NewLeafID())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- GetDescendants tests ---

func TestGetDescendants_ReturnsLeaves(t *testing.T) {
	child := makeLeaf("child")

	graph := &stubGraphEngine{
		descendantsFn: func(_ context.Context, _ types.LeafID) ([]domain.Leaf, error) {
			return []domain.Leaf{child}, nil
		},
	}
	svc := newTestService(graph)

	leaves, err := svc.GetDescendants(context.Background(), types.NewLeafID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(leaves) != 1 {
		t.Fatalf("got %d leaves, want 1", len(leaves))
	}
	if leaves[0].Title() != "child" {
		t.Errorf("descendant title = %q, want %q", leaves[0].Title(), "child")
	}
}

func TestGetDescendants_WrapsError(t *testing.T) {
	graph := &stubGraphEngine{
		descendantsFn: func(_ context.Context, _ types.LeafID) ([]domain.Leaf, error) {
			return nil, fmt.Errorf("query timeout")
		},
	}
	svc := newTestService(graph)

	_, err := svc.GetDescendants(context.Background(), types.NewLeafID())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- GetNeighborhood tests ---

func TestGetNeighborhood_ReturnsLeaves(t *testing.T) {
	neighbor := makeLeaf("neighbor")

	graph := &stubGraphEngine{
		neighborhoodFn: func(_ context.Context, _ types.LeafID, depth int) ([]domain.Leaf, error) {
			if depth != 3 {
				t.Errorf("depth = %d, want 3", depth)
			}
			return []domain.Leaf{neighbor}, nil
		},
	}
	svc := newTestService(graph)

	leaves, err := svc.GetNeighborhood(context.Background(), types.NewLeafID(), 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(leaves) != 1 {
		t.Fatalf("got %d leaves, want 1", len(leaves))
	}
}

func TestGetNeighborhood_WrapsError(t *testing.T) {
	graph := &stubGraphEngine{
		neighborhoodFn: func(_ context.Context, _ types.LeafID, _ int) ([]domain.Leaf, error) {
			return nil, fmt.Errorf("too many connections")
		},
	}
	svc := newTestService(graph)

	_, err := svc.GetNeighborhood(context.Background(), types.NewLeafID(), 2)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- GetSynthesisLineage tests ---

func TestGetSynthesisLineage_ReturnsLeaves(t *testing.T) {
	sourceA := makeLeaf("source A")
	sourceB := makeLeaf("source B")

	graph := &stubGraphEngine{
		lineageFn: func(_ context.Context, _ types.LeafID) ([]domain.Leaf, error) {
			return []domain.Leaf{sourceA, sourceB}, nil
		},
	}
	svc := newTestService(graph)

	leaves, err := svc.GetSynthesisLineage(context.Background(), types.NewLeafID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(leaves) != 2 {
		t.Fatalf("got %d leaves, want 2", len(leaves))
	}
}

func TestGetSynthesisLineage_EmptyForNonSynthesis(t *testing.T) {
	graph := &stubGraphEngine{
		lineageFn: func(_ context.Context, _ types.LeafID) ([]domain.Leaf, error) {
			return nil, nil
		},
	}
	svc := newTestService(graph)

	leaves, err := svc.GetSynthesisLineage(context.Background(), types.NewLeafID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(leaves) != 0 {
		t.Fatalf("got %d leaves, want 0", len(leaves))
	}
}

func TestGetSynthesisLineage_WrapsError(t *testing.T) {
	graph := &stubGraphEngine{
		lineageFn: func(_ context.Context, _ types.LeafID) ([]domain.Leaf, error) {
			return nil, fmt.Errorf("lineage broken")
		},
	}
	svc := newTestService(graph)

	_, err := svc.GetSynthesisLineage(context.Background(), types.NewLeafID())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- PassesCorrectLeafID tests ---

func TestGetAncestors_PassesCorrectLeafID(t *testing.T) {
	targetID := types.NewLeafID()
	var receivedID types.LeafID

	graph := &stubGraphEngine{
		ancestorsFn: func(_ context.Context, leafID types.LeafID) ([]domain.Leaf, error) {
			receivedID = leafID
			return nil, nil
		},
	}
	svc := newTestService(graph)

	_, _ = svc.GetAncestors(context.Background(), targetID)
	if receivedID != targetID {
		t.Errorf("received leaf ID = %s, want %s", receivedID, targetID)
	}
}
