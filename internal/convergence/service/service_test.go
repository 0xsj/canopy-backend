package service

import (
	"context"
	"testing"

	"github.com/0xsj/canopy-backend/internal/convergence/domain"
	wsdomain "github.com/0xsj/canopy-backend/internal/workspace/domain"
	"github.com/0xsj/canopy-backend/pkg/auth"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// --- Test doubles ---

type stubSignalRepo struct {
	createFn       func(ctx context.Context, signal domain.Signal) error
	deleteFn       func(ctx context.Context, id domain.SignalID) error
	findByLeafFn   func(ctx context.Context, leafID types.LeafID) ([]domain.Signal, error)
	findByUserFn   func(ctx context.Context, wsID types.WorkspaceID, userID types.UserID) ([]domain.Signal, error)
	countByLeafFn  func(ctx context.Context, leafID types.LeafID) (map[domain.SignalType]int, error)
	createdSignals []domain.Signal
}

func (s *stubSignalRepo) Create(ctx context.Context, signal domain.Signal) error {
	s.createdSignals = append(s.createdSignals, signal)
	if s.createFn != nil {
		return s.createFn(ctx, signal)
	}
	return nil
}

func (s *stubSignalRepo) Delete(ctx context.Context, id domain.SignalID) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, id)
	}
	return nil
}

func (s *stubSignalRepo) FindByLeaf(ctx context.Context, leafID types.LeafID) ([]domain.Signal, error) {
	if s.findByLeafFn != nil {
		return s.findByLeafFn(ctx, leafID)
	}
	return nil, nil
}

func (s *stubSignalRepo) FindByUser(ctx context.Context, wsID types.WorkspaceID, userID types.UserID) ([]domain.Signal, error) {
	if s.findByUserFn != nil {
		return s.findByUserFn(ctx, wsID, userID)
	}
	return nil, nil
}

func (s *stubSignalRepo) FindByLeafAndUser(_ context.Context, _ types.LeafID, _ types.UserID) (domain.Signal, error) {
	return domain.Signal{}, canopyerr.ErrNotFound
}

func (s *stubSignalRepo) CountByLeaf(ctx context.Context, leafID types.LeafID) (map[domain.SignalType]int, error) {
	if s.countByLeafFn != nil {
		return s.countByLeafFn(ctx, leafID)
	}
	return map[domain.SignalType]int{}, nil
}

type stubCheckpointRepo struct {
	createFn          func(ctx context.Context, cp domain.Checkpoint) error
	findByIDFn        func(ctx context.Context, id types.CheckpointID) (domain.Checkpoint, error)
	findByWSFn        func(ctx context.Context, wsID types.WorkspaceID) ([]domain.Checkpoint, error)
	findOpenByWSFn    func(ctx context.Context, wsID types.WorkspaceID) ([]domain.Checkpoint, error)
	updateFn          func(ctx context.Context, cp domain.Checkpoint) error
	createdCheckpoint domain.Checkpoint
	updatedCheckpoint domain.Checkpoint
	updateCalls       int
}

func (s *stubCheckpointRepo) Create(ctx context.Context, cp domain.Checkpoint) error {
	s.createdCheckpoint = cp
	if s.createFn != nil {
		return s.createFn(ctx, cp)
	}
	return nil
}

func (s *stubCheckpointRepo) FindByID(ctx context.Context, id types.CheckpointID) (domain.Checkpoint, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(ctx, id)
	}
	return domain.Checkpoint{}, canopyerr.ErrNotFound
}

func (s *stubCheckpointRepo) FindByWorkspace(ctx context.Context, wsID types.WorkspaceID) ([]domain.Checkpoint, error) {
	if s.findByWSFn != nil {
		return s.findByWSFn(ctx, wsID)
	}
	return nil, nil
}

func (s *stubCheckpointRepo) FindOpenByWorkspace(ctx context.Context, wsID types.WorkspaceID) ([]domain.Checkpoint, error) {
	if s.findOpenByWSFn != nil {
		return s.findOpenByWSFn(ctx, wsID)
	}
	return nil, nil
}

func (s *stubCheckpointRepo) Update(ctx context.Context, cp domain.Checkpoint) error {
	s.updatedCheckpoint = cp
	s.updateCalls++
	if s.updateFn != nil {
		return s.updateFn(ctx, cp)
	}
	return nil
}

type stubLeafWriter struct {
	updateLayerFn func(ctx context.Context, id types.LeafID, layer string) error
	promotedLeafs []types.LeafID
}

func (s *stubLeafWriter) UpdateLayer(ctx context.Context, id types.LeafID, layer string) error {
	s.promotedLeafs = append(s.promotedLeafs, id)
	if s.updateLayerFn != nil {
		return s.updateLayerFn(ctx, id, layer)
	}
	return nil
}

type stubWSMembers struct {
	findMemberFn func(ctx context.Context, wsID types.WorkspaceID, userID types.UserID) (wsdomain.WorkspaceMember, error)
}

func (s *stubWSMembers) FindMember(ctx context.Context, wsID types.WorkspaceID, userID types.UserID) (wsdomain.WorkspaceMember, error) {
	if s.findMemberFn != nil {
		return s.findMemberFn(ctx, wsID, userID)
	}
	return wsdomain.WorkspaceMember{}, nil
}

type stubPublisher struct {
	published    []events.Event
	publishCalls int
}

func (s *stubPublisher) Publish(_ context.Context, event events.Event) error {
	s.published = append(s.published, event)
	s.publishCalls++
	return nil
}

// --- Interface conformance ---

var _ domain.SignalRepository = (*stubSignalRepo)(nil)
var _ domain.CheckpointRepository = (*stubCheckpointRepo)(nil)
var _ LeafWriter = (*stubLeafWriter)(nil)
var _ WorkspaceMemberReader = (*stubWSMembers)(nil)
var _ events.Publisher = (*stubPublisher)(nil)

// --- Test helpers ---

func testCtx(userID string) context.Context {
	claims := auth.Claims{Subject: userID, Issuer: "test"}
	return auth.WithClaims(context.Background(), claims)
}

func testCtxNoAuth() context.Context {
	return context.Background()
}

func newTestService(
	signals *stubSignalRepo,
	checkpoints *stubCheckpointRepo,
	leafWriter *stubLeafWriter,
	wsMembers *stubWSMembers,
	pub *stubPublisher,
) *Service {
	return New(signals, checkpoints, leafWriter, wsMembers, pub, logger.NewNoop())
}

func openCheckpoint(wsID types.WorkspaceID, leafIDs ...types.LeafID) domain.Checkpoint {
	cp, _ := domain.NewCheckpoint(wsID, leafIDs)
	return cp
}

// --- RecordConsensusPosition tests ---

func TestRecordConsensusPosition_HappyPath(t *testing.T) {
	wsID := types.NewWorkspaceID()
	cp := openCheckpoint(wsID, types.NewLeafID())

	checkpoints := &stubCheckpointRepo{
		findByIDFn: func(_ context.Context, _ types.CheckpointID) (domain.Checkpoint, error) {
			return cp, nil
		},
	}
	pub := &stubPublisher{}
	svc := newTestService(&stubSignalRepo{}, checkpoints, &stubLeafWriter{}, &stubWSMembers{}, pub)

	result, err := svc.RecordConsensusPosition(testCtx("user_alice"), cp.ID(), domain.PositionAlign, "looks good")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Signals()) != 1 {
		t.Fatalf("signals count = %d, want 1", len(result.Signals()))
	}
	if result.Signals()[0].Position() != domain.PositionAlign {
		t.Errorf("position = %q, want %q", result.Signals()[0].Position(), domain.PositionAlign)
	}
	if checkpoints.updateCalls != 1 {
		t.Errorf("update calls = %d, want 1", checkpoints.updateCalls)
	}
	if pub.publishCalls != 1 {
		t.Errorf("publish calls = %d, want 1", pub.publishCalls)
	}
}

func TestRecordConsensusPosition_NoAuth(t *testing.T) {
	wsID := types.NewWorkspaceID()
	cp := openCheckpoint(wsID, types.NewLeafID())

	checkpoints := &stubCheckpointRepo{
		findByIDFn: func(_ context.Context, _ types.CheckpointID) (domain.Checkpoint, error) {
			return cp, nil
		},
	}
	svc := newTestService(&stubSignalRepo{}, checkpoints, &stubLeafWriter{}, &stubWSMembers{}, &stubPublisher{})

	_, err := svc.RecordConsensusPosition(testCtxNoAuth(), cp.ID(), domain.PositionAlign, "")
	if err == nil {
		t.Fatal("expected error when no auth claims")
	}
	if !canopyerr.Is(err, canopyerr.ErrUnauthenticated) {
		t.Errorf("expected unauthenticated error, got: %v", err)
	}
}

func TestRecordConsensusPosition_NotMember(t *testing.T) {
	wsID := types.NewWorkspaceID()
	cp := openCheckpoint(wsID, types.NewLeafID())

	checkpoints := &stubCheckpointRepo{
		findByIDFn: func(_ context.Context, _ types.CheckpointID) (domain.Checkpoint, error) {
			return cp, nil
		},
	}
	wsMembers := &stubWSMembers{
		findMemberFn: func(_ context.Context, _ types.WorkspaceID, _ types.UserID) (wsdomain.WorkspaceMember, error) {
			return wsdomain.WorkspaceMember{}, canopyerr.ErrNotFound
		},
	}
	svc := newTestService(&stubSignalRepo{}, checkpoints, &stubLeafWriter{}, wsMembers, &stubPublisher{})

	_, err := svc.RecordConsensusPosition(testCtx("user_outsider"), cp.ID(), domain.PositionAlign, "")
	if err == nil {
		t.Fatal("expected error for non-member")
	}
	if !canopyerr.Is(err, canopyerr.ErrUnauthorized) {
		t.Errorf("expected unauthorized error, got: %v", err)
	}
}

func TestRecordConsensusPosition_InvalidPosition(t *testing.T) {
	wsID := types.NewWorkspaceID()
	cp := openCheckpoint(wsID, types.NewLeafID())

	checkpoints := &stubCheckpointRepo{
		findByIDFn: func(_ context.Context, _ types.CheckpointID) (domain.Checkpoint, error) {
			return cp, nil
		},
	}
	svc := newTestService(&stubSignalRepo{}, checkpoints, &stubLeafWriter{}, &stubWSMembers{}, &stubPublisher{})

	_, err := svc.RecordConsensusPosition(testCtx("user_alice"), cp.ID(), domain.Position("invalid"), "")
	if err == nil {
		t.Fatal("expected error for invalid position")
	}
}

func TestRecordConsensusPosition_DuplicateUser(t *testing.T) {
	wsID := types.NewWorkspaceID()
	cp := openCheckpoint(wsID, types.NewLeafID())

	// Pre-record a signal from user_alice on the checkpoint.
	sig, _ := domain.NewConsensusSignal(cp.ID(), types.UserIDFrom("user_alice"), domain.PositionAlign, "")
	_ = cp.RecordSignal(sig)

	checkpoints := &stubCheckpointRepo{
		findByIDFn: func(_ context.Context, _ types.CheckpointID) (domain.Checkpoint, error) {
			return cp, nil
		},
	}
	svc := newTestService(&stubSignalRepo{}, checkpoints, &stubLeafWriter{}, &stubWSMembers{}, &stubPublisher{})

	_, err := svc.RecordConsensusPosition(testCtx("user_alice"), cp.ID(), domain.PositionConcern, "changed mind")
	if err == nil {
		t.Fatal("expected error for duplicate signal from same user")
	}
}

func TestRecordConsensusPosition_ResolvedCheckpoint(t *testing.T) {
	wsID := types.NewWorkspaceID()
	cp := openCheckpoint(wsID, types.NewLeafID())
	_ = cp.Resolve()

	checkpoints := &stubCheckpointRepo{
		findByIDFn: func(_ context.Context, _ types.CheckpointID) (domain.Checkpoint, error) {
			return cp, nil
		},
	}
	svc := newTestService(&stubSignalRepo{}, checkpoints, &stubLeafWriter{}, &stubWSMembers{}, &stubPublisher{})

	_, err := svc.RecordConsensusPosition(testCtx("user_alice"), cp.ID(), domain.PositionAlign, "")
	if err == nil {
		t.Fatal("expected error for resolved checkpoint")
	}
}

func TestRecordConsensusPosition_CheckpointNotFound(t *testing.T) {
	svc := newTestService(&stubSignalRepo{}, &stubCheckpointRepo{}, &stubLeafWriter{}, &stubWSMembers{}, &stubPublisher{})

	_, err := svc.RecordConsensusPosition(testCtx("user_alice"), types.NewCheckpointID(), domain.PositionAlign, "")
	if err == nil {
		t.Fatal("expected error for missing checkpoint")
	}
	if !canopyerr.Is(err, canopyerr.ErrNotFound) {
		t.Errorf("expected not found error, got: %v", err)
	}
}

// --- GetCheckpoint tests ---

func TestGetCheckpoint_HappyPath(t *testing.T) {
	wsID := types.NewWorkspaceID()
	cp := openCheckpoint(wsID, types.NewLeafID())

	checkpoints := &stubCheckpointRepo{
		findByIDFn: func(_ context.Context, _ types.CheckpointID) (domain.Checkpoint, error) {
			return cp, nil
		},
	}
	svc := newTestService(&stubSignalRepo{}, checkpoints, &stubLeafWriter{}, &stubWSMembers{}, &stubPublisher{})

	result, err := svc.GetCheckpoint(testCtx("user_alice"), cp.ID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID() != cp.ID() {
		t.Errorf("id = %q, want %q", result.ID(), cp.ID())
	}
}

func TestGetCheckpoint_NotFound(t *testing.T) {
	svc := newTestService(&stubSignalRepo{}, &stubCheckpointRepo{}, &stubLeafWriter{}, &stubWSMembers{}, &stubPublisher{})

	_, err := svc.GetCheckpoint(testCtx("user_alice"), types.NewCheckpointID())
	if err == nil {
		t.Fatal("expected error for missing checkpoint")
	}
	if !canopyerr.Is(err, canopyerr.ErrNotFound) {
		t.Errorf("expected not found error, got: %v", err)
	}
}

// --- FindCheckpointsByWorkspace tests ---

func TestFindCheckpointsByWorkspace_All(t *testing.T) {
	wsID := types.NewWorkspaceID()
	cp1 := openCheckpoint(wsID, types.NewLeafID())
	cp2 := openCheckpoint(wsID, types.NewLeafID())
	_ = cp2.Resolve()

	checkpoints := &stubCheckpointRepo{
		findByWSFn: func(_ context.Context, _ types.WorkspaceID) ([]domain.Checkpoint, error) {
			return []domain.Checkpoint{cp1, cp2}, nil
		},
	}
	svc := newTestService(&stubSignalRepo{}, checkpoints, &stubLeafWriter{}, &stubWSMembers{}, &stubPublisher{})

	result, err := svc.FindCheckpointsByWorkspace(testCtx("user_alice"), wsID, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("count = %d, want 2", len(result))
	}
}

func TestFindCheckpointsByWorkspace_OpenOnly(t *testing.T) {
	wsID := types.NewWorkspaceID()
	cp := openCheckpoint(wsID, types.NewLeafID())

	checkpoints := &stubCheckpointRepo{
		findOpenByWSFn: func(_ context.Context, _ types.WorkspaceID) ([]domain.Checkpoint, error) {
			return []domain.Checkpoint{cp}, nil
		},
	}
	svc := newTestService(&stubSignalRepo{}, checkpoints, &stubLeafWriter{}, &stubWSMembers{}, &stubPublisher{})

	result, err := svc.FindCheckpointsByWorkspace(testCtx("user_alice"), wsID, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("count = %d, want 1", len(result))
	}
	if result[0].Status() != domain.CheckpointOpen {
		t.Errorf("status = %q, want %q", result[0].Status(), domain.CheckpointOpen)
	}
}

func TestFindCheckpointsByWorkspace_NoAuth(t *testing.T) {
	svc := newTestService(&stubSignalRepo{}, &stubCheckpointRepo{}, &stubLeafWriter{}, &stubWSMembers{}, &stubPublisher{})

	_, err := svc.FindCheckpointsByWorkspace(testCtxNoAuth(), types.NewWorkspaceID(), false)
	if err == nil {
		t.Fatal("expected error when no auth claims")
	}
	if !canopyerr.Is(err, canopyerr.ErrUnauthenticated) {
		t.Errorf("expected unauthenticated error, got: %v", err)
	}
}

// --- ResolveCheckpoint with consensus tests ---

func TestResolveCheckpoint_WithConsensus_PromotesLeaves(t *testing.T) {
	wsID := types.NewWorkspaceID()
	leaf1 := types.NewLeafID()
	leaf2 := types.NewLeafID()
	cp := openCheckpoint(wsID, leaf1, leaf2)

	// Two participants aligned.
	sig1, _ := domain.NewConsensusSignal(cp.ID(), types.NewUserID(), domain.PositionAlign, "")
	sig2, _ := domain.NewConsensusSignal(cp.ID(), types.NewUserID(), domain.PositionAlign, "")
	_ = cp.RecordSignal(sig1)
	_ = cp.RecordSignal(sig2)

	checkpoints := &stubCheckpointRepo{
		findByIDFn: func(_ context.Context, _ types.CheckpointID) (domain.Checkpoint, error) {
			return cp, nil
		},
	}
	leafWriter := &stubLeafWriter{}
	pub := &stubPublisher{}
	svc := newTestService(&stubSignalRepo{}, checkpoints, leafWriter, &stubWSMembers{}, pub)

	err := svc.ResolveCheckpoint(testCtx("user_alice"), cp.ID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(leafWriter.promotedLeafs) != 2 {
		t.Errorf("promoted = %d, want 2", len(leafWriter.promotedLeafs))
	}
	if checkpoints.updateCalls != 1 {
		t.Errorf("update calls = %d, want 1", checkpoints.updateCalls)
	}
	// 2 leaf.promoted + 1 consensus.reached = 3 events.
	if pub.publishCalls != 3 {
		t.Errorf("publish calls = %d, want 3", pub.publishCalls)
	}
}

func TestResolveCheckpoint_WithoutConsensus_NoPromotion(t *testing.T) {
	wsID := types.NewWorkspaceID()
	cp := openCheckpoint(wsID, types.NewLeafID())

	// One align + one block = no consensus.
	sig1, _ := domain.NewConsensusSignal(cp.ID(), types.NewUserID(), domain.PositionAlign, "")
	sig2, _ := domain.NewConsensusSignal(cp.ID(), types.NewUserID(), domain.PositionBlock, "disagree")
	_ = cp.RecordSignal(sig1)
	_ = cp.RecordSignal(sig2)

	checkpoints := &stubCheckpointRepo{
		findByIDFn: func(_ context.Context, _ types.CheckpointID) (domain.Checkpoint, error) {
			return cp, nil
		},
	}
	leafWriter := &stubLeafWriter{}
	pub := &stubPublisher{}
	svc := newTestService(&stubSignalRepo{}, checkpoints, leafWriter, &stubWSMembers{}, pub)

	err := svc.ResolveCheckpoint(testCtx("user_alice"), cp.ID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(leafWriter.promotedLeafs) != 0 {
		t.Errorf("promoted = %d, want 0", len(leafWriter.promotedLeafs))
	}
	// No promotion events, no consensus.reached event.
	if pub.publishCalls != 0 {
		t.Errorf("publish calls = %d, want 0", pub.publishCalls)
	}
}

func TestResolveCheckpoint_AlreadyResolved(t *testing.T) {
	wsID := types.NewWorkspaceID()
	cp := openCheckpoint(wsID, types.NewLeafID())
	_ = cp.Resolve()

	checkpoints := &stubCheckpointRepo{
		findByIDFn: func(_ context.Context, _ types.CheckpointID) (domain.Checkpoint, error) {
			return cp, nil
		},
	}
	svc := newTestService(&stubSignalRepo{}, checkpoints, &stubLeafWriter{}, &stubWSMembers{}, &stubPublisher{})

	err := svc.ResolveCheckpoint(testCtx("user_alice"), cp.ID())
	if err == nil {
		t.Fatal("expected error for already resolved checkpoint")
	}
}
