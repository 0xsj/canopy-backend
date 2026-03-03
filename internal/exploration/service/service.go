package service

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/0xsj/canopy-backend/internal/exploration/domain"
	wsdomain "github.com/0xsj/canopy-backend/internal/workspace/domain"
	"github.com/0xsj/canopy-backend/pkg/auth"
	"github.com/0xsj/canopy-backend/pkg/database"
	canopyerr "github.com/0xsj/canopy-backend/pkg/errors"
	"github.com/0xsj/canopy-backend/pkg/events"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// WorkspaceMemberReader is a cross-context read port for workspace membership.
type WorkspaceMemberReader interface {
	FindMember(ctx context.Context, workspaceID types.WorkspaceID, userID types.UserID) (wsdomain.WorkspaceMember, error)
}

// Service implements the exploration application logic.
type Service struct {
	leaves        domain.LeafRepository
	branches      domain.BranchRepository
	connections   domain.ConnectionRepository
	graph         domain.GraphQueryEngine
	wsMembers     WorkspaceMemberReader
	db            *database.DB
	newLeafRepo   func(database.DBTX) domain.LeafRepository
	newBranchRepo func(database.DBTX) domain.BranchRepository
	pub           events.Publisher
	log           logger.Logger
}

// New creates a new exploration service.
func New(
	leaves domain.LeafRepository,
	branches domain.BranchRepository,
	connections domain.ConnectionRepository,
	graph domain.GraphQueryEngine,
	wsMembers WorkspaceMemberReader,
	db *database.DB,
	newLeafRepo func(database.DBTX) domain.LeafRepository,
	newBranchRepo func(database.DBTX) domain.BranchRepository,
	pub events.Publisher,
	log logger.Logger,
) *Service {
	return &Service{
		leaves:        leaves,
		branches:      branches,
		connections:   connections,
		graph:         graph,
		wsMembers:     wsMembers,
		db:            db,
		newLeafRepo:   newLeafRepo,
		newBranchRepo: newBranchRepo,
		pub:           pub,
		log:           log,
	}
}

// CreateLeaf creates a new leaf on an existing branch. Caller must be a workspace member.
func (s *Service) CreateLeaf(
	ctx context.Context,
	workspaceID types.WorkspaceID,
	seedID types.SeedID,
	branchID types.BranchID,
	parentLeafID types.LeafID,
	title, summary string,
	keyPoints, openQuestions, tags []string,
) (domain.Leaf, error) {
	const op = "exploration: create leaf"

	callerID, err := s.requireMember(ctx, workspaceID)
	if err != nil {
		return domain.Leaf{}, canopyerr.Wrap(err, op)
	}

	leaf, err := domain.NewLeaf(workspaceID, seedID, branchID, callerID, parentLeafID, title, summary, keyPoints, openQuestions, tags)
	if err != nil {
		return domain.Leaf{}, canopyerr.Wrap(err, op)
	}

	if err := s.leaves.Create(ctx, leaf); err != nil {
		return domain.Leaf{}, canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectLeafCreated, workspaceID.String(), domain.LeafCreatedData{
		LeafID:      leaf.ID().String(),
		WorkspaceID: workspaceID.String(),
		SeedID:      seedID.String(),
		BranchID:    branchID.String(),
		AuthorID:    callerID.String(),
		ParentID:    parentLeafID.String(),
		Title:       leaf.Title(),
		Layer:       string(leaf.Layer()),
		IsSynthesis: leaf.IsSynthesis(),
		Timestamp:   time.Now().UTC(),
	})

	s.log.Info("leaf created",
		logger.String("leaf_id", leaf.ID().String()),
		logger.String("workspace_id", workspaceID.String()),
	)

	return leaf, nil
}

// FindLeafByID returns a leaf by ID.
func (s *Service) FindLeafByID(ctx context.Context, id types.LeafID) (domain.Leaf, error) {
	const op = "exploration: find leaf by id"
	leaf, err := s.leaves.FindByID(ctx, id)
	if err != nil {
		return domain.Leaf{}, canopyerr.Wrap(err, op)
	}
	return leaf, nil
}

// FindLeavesByWorkspace returns leaves in a workspace with optional filters.
func (s *Service) FindLeavesByWorkspace(ctx context.Context, workspaceID types.WorkspaceID, filter domain.LeafFilter) ([]domain.Leaf, error) {
	const op = "exploration: find leaves by workspace"

	if _, err := s.requireMember(ctx, workspaceID); err != nil {
		return nil, canopyerr.Wrap(err, op)
	}

	leaves, err := s.leaves.FindByWorkspace(ctx, workspaceID, filter)
	if err != nil {
		return nil, canopyerr.Wrap(err, op)
	}
	return leaves, nil
}

// SearchLeaves performs full-text search across leaf content within a workspace.
func (s *Service) SearchLeaves(ctx context.Context, workspaceID types.WorkspaceID, query string) ([]domain.Leaf, error) {
	const op = "exploration: search leaves"

	if _, err := s.requireMember(ctx, workspaceID); err != nil {
		return nil, canopyerr.Wrap(err, op)
	}

	leaves, err := s.leaves.Search(ctx, workspaceID, query)
	if err != nil {
		return nil, canopyerr.Wrap(err, op)
	}
	return leaves, nil
}

// StartBranch creates a branch and its first leaf atomically.
func (s *Service) StartBranch(
	ctx context.Context,
	workspaceID types.WorkspaceID,
	seedID types.SeedID,
	parentLeafID types.LeafID,
	title, summary string,
	keyPoints, openQuestions, tags []string,
) (domain.Branch, domain.Leaf, error) {
	const op = "exploration: start branch"

	callerID, err := s.requireMember(ctx, workspaceID)
	if err != nil {
		return domain.Branch{}, domain.Leaf{}, canopyerr.Wrap(err, op)
	}

	branch, err := domain.NewBranch(workspaceID, seedID, callerID)
	if err != nil {
		return domain.Branch{}, domain.Leaf{}, canopyerr.Wrap(err, op)
	}

	leaf, err := domain.NewLeaf(workspaceID, seedID, branch.ID(), callerID, parentLeafID, title, summary, keyPoints, openQuestions, tags)
	if err != nil {
		return domain.Branch{}, domain.Leaf{}, canopyerr.Wrap(err, op)
	}

	if err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		txBranches := s.newBranchRepo(tx)
		txLeaves := s.newLeafRepo(tx)
		// Insert branch without root_leaf_id first (leaf FK references branch).
		if err := txBranches.Create(ctx, branch); err != nil {
			return err
		}
		// Insert leaf (references branch via FK).
		if err := txLeaves.Create(ctx, leaf); err != nil {
			return err
		}
		// Now set root_leaf_id on branch (leaf exists, FK satisfied).
		if err := branch.SetRootLeaf(leaf.ID()); err != nil {
			return err
		}
		return txBranches.Update(ctx, branch)
	}); err != nil {
		return domain.Branch{}, domain.Leaf{}, canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectBranchCreated, workspaceID.String(), domain.BranchCreatedData{
		BranchID:    branch.ID().String(),
		WorkspaceID: workspaceID.String(),
		SeedID:      seedID.String(),
		AuthorID:    callerID.String(),
		Timestamp:   time.Now().UTC(),
	})

	s.publish(ctx, domain.SubjectLeafCreated, workspaceID.String(), domain.LeafCreatedData{
		LeafID:      leaf.ID().String(),
		WorkspaceID: workspaceID.String(),
		SeedID:      seedID.String(),
		BranchID:    branch.ID().String(),
		AuthorID:    callerID.String(),
		Title:       leaf.Title(),
		Layer:       string(leaf.Layer()),
		Timestamp:   time.Now().UTC(),
	})

	s.log.Info("branch started",
		logger.String("branch_id", branch.ID().String()),
		logger.String("leaf_id", leaf.ID().String()),
	)

	return branch, leaf, nil
}

// FindBranchByID returns a branch by ID.
func (s *Service) FindBranchByID(ctx context.Context, id types.BranchID) (domain.Branch, error) {
	const op = "exploration: find branch by id"
	branch, err := s.branches.FindByID(ctx, id)
	if err != nil {
		return domain.Branch{}, canopyerr.Wrap(err, op)
	}
	return branch, nil
}

// CreateConnection creates a connection between leaves. Caller must be a workspace member.
func (s *Service) CreateConnection(ctx context.Context, workspaceID types.WorkspaceID, leafIDs []types.LeafID) (domain.Connection, error) {
	const op = "exploration: create connection"

	callerID, err := s.requireMember(ctx, workspaceID)
	if err != nil {
		return domain.Connection{}, canopyerr.Wrap(err, op)
	}

	conn, err := domain.NewConnection(workspaceID, callerID, leafIDs)
	if err != nil {
		return domain.Connection{}, canopyerr.Wrap(err, op)
	}

	if err := s.connections.Create(ctx, conn); err != nil {
		return domain.Connection{}, canopyerr.Wrap(err, op)
	}

	leafIDStrs := make([]string, len(leafIDs))
	for i, id := range leafIDs {
		leafIDStrs[i] = id.String()
	}

	s.publish(ctx, domain.SubjectConnectionCreated, workspaceID.String(), domain.ConnectionCreatedData{
		ConnectionID: conn.ID().String(),
		WorkspaceID:  workspaceID.String(),
		AuthorID:     callerID.String(),
		LeafIDs:      leafIDStrs,
		Timestamp:    time.Now().UTC(),
	})

	return conn, nil
}

// FindConnectionsByWorkspace returns all connections in a workspace.
func (s *Service) FindConnectionsByWorkspace(ctx context.Context, workspaceID types.WorkspaceID) ([]domain.Connection, error) {
	const op = "exploration: find connections by workspace"

	if _, err := s.requireMember(ctx, workspaceID); err != nil {
		return nil, canopyerr.Wrap(err, op)
	}

	conns, err := s.connections.FindByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, canopyerr.Wrap(err, op)
	}
	return conns, nil
}

// FindConnectionsByLeaf returns all connections that include a given leaf.
func (s *Service) FindConnectionsByLeaf(ctx context.Context, leafID types.LeafID) ([]domain.Connection, error) {
	const op = "exploration: find connections by leaf"
	conns, err := s.connections.FindByLeaf(ctx, leafID)
	if err != nil {
		return nil, canopyerr.Wrap(err, op)
	}
	return conns, nil
}

// PromoteLeaf promotes a leaf from understory to canopy layer.
// This is called by the convergence context when consensus is reached.
func (s *Service) PromoteLeaf(ctx context.Context, leafID types.LeafID) error {
	const op = "exploration: promote leaf"

	leaf, err := s.leaves.FindByID(ctx, leafID)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := s.leaves.UpdateLayer(ctx, leafID, domain.LayerCanopy); err != nil {
		return canopyerr.Wrap(err, op)
	}

	s.publish(ctx, domain.SubjectLeafPromoted, leaf.WorkspaceID().String(), domain.LeafPromotedData{
		LeafID:      leafID.String(),
		WorkspaceID: leaf.WorkspaceID().String(),
		Timestamp:   time.Now().UTC(),
	})

	return nil
}

// UpdateLeafPosition updates a leaf's canvas position. Caller must be a workspace member.
func (s *Service) UpdateLeafPosition(ctx context.Context, leafID types.LeafID, x, y float64) error {
	const op = "exploration: update leaf position"

	leaf, err := s.leaves.FindByID(ctx, leafID)
	if err != nil {
		return canopyerr.Wrap(err, op)
	}

	if _, err := s.requireMember(ctx, leaf.WorkspaceID()); err != nil {
		return canopyerr.Wrap(err, op)
	}

	if err := s.leaves.UpdatePosition(ctx, leafID, x, y); err != nil {
		return canopyerr.Wrap(err, op)
	}

	return nil
}

// --- Graph Queries ---

// GetAncestors returns all ancestor leaves back to the seed root.
func (s *Service) GetAncestors(ctx context.Context, leafID types.LeafID) ([]domain.Leaf, error) {
	const op = "exploration: get ancestors"
	leaves, err := s.graph.Ancestors(ctx, leafID)
	if err != nil {
		return nil, canopyerr.Wrap(err, op)
	}
	return leaves, nil
}

// GetDescendants returns all descendant leaves from a given leaf.
func (s *Service) GetDescendants(ctx context.Context, leafID types.LeafID) ([]domain.Leaf, error) {
	const op = "exploration: get descendants"
	leaves, err := s.graph.Descendants(ctx, leafID)
	if err != nil {
		return nil, canopyerr.Wrap(err, op)
	}
	return leaves, nil
}

// GetNeighborhood returns all leaves within N connections of a given leaf.
func (s *Service) GetNeighborhood(ctx context.Context, leafID types.LeafID, depth int) ([]domain.Leaf, error) {
	const op = "exploration: get neighborhood"
	leaves, err := s.graph.Neighborhood(ctx, leafID, depth)
	if err != nil {
		return nil, canopyerr.Wrap(err, op)
	}
	return leaves, nil
}

// GetSynthesisLineage traces a synthesis leaf back through its source chain.
func (s *Service) GetSynthesisLineage(ctx context.Context, leafID types.LeafID) ([]domain.Leaf, error) {
	const op = "exploration: get synthesis lineage"
	leaves, err := s.graph.SynthesisLineage(ctx, leafID)
	if err != nil {
		return nil, canopyerr.Wrap(err, op)
	}
	return leaves, nil
}

// --- Auth Helpers ---

func (s *Service) requireMember(ctx context.Context, workspaceID types.WorkspaceID) (types.UserID, error) {
	claims, ok := auth.FromClaims(ctx)
	if !ok {
		return types.UserID{}, canopyerr.ErrUnauthenticated
	}
	callerID := types.UserIDFrom(claims.Subject)

	if _, err := s.wsMembers.FindMember(ctx, workspaceID, callerID); err != nil {
		if canopyerr.GetKind(err) == canopyerr.KindNotFound {
			return types.UserID{}, canopyerr.ErrUnauthorized
		}
		return types.UserID{}, err
	}
	return callerID, nil
}

// --- Event Publishing ---

func (s *Service) publish(ctx context.Context, eventType, workspaceID string, data any) {
	event, err := events.New(eventType, workspaceID, data)
	if err != nil {
		s.log.Error("event creation failed", logger.String("type", eventType), logger.Err(err))
		return
	}
	event.Subject = events.BuildSubject(workspaceID, "exploration", eventType)
	if pubErr := s.pub.Publish(ctx, event); pubErr != nil {
		s.log.Error("event publish failed", logger.String("type", eventType), logger.Err(pubErr))
	}
}
