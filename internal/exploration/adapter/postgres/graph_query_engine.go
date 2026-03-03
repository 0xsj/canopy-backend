package postgres

import (
	"context"
	"fmt"

	"github.com/0xsj/canopy-backend/internal/exploration/adapter/postgres/sqlc"
	"github.com/0xsj/canopy-backend/internal/exploration/domain"
	"github.com/0xsj/canopy-backend/pkg/database"
	"github.com/0xsj/canopy-backend/pkg/types"
)

// GraphQueryEngine implements domain.GraphQueryEngine using Postgres
// recursive CTEs and Go-level BFS traversal.
type GraphQueryEngine struct {
	db database.DBTX
}

// NewGraphQueryEngine creates a new GraphQueryEngine.
func NewGraphQueryEngine(db database.DBTX) *GraphQueryEngine {
	return &GraphQueryEngine{db: db}
}

var _ domain.GraphQueryEngine = (*GraphQueryEngine)(nil)

// leafCols is the unqualified column list matching sqlc.Leafe field order.
const leafCols = `id, workspace_id, seed_id, branch_id, author_id, parent_leaf_id,
    title, summary, key_points, open_questions, tags, layer, sources, metadata, created_at, position_x, position_y, search_vector`

// leafColsL is the same column list qualified with table alias "l." for use in JOINs.
const leafColsL = `l.id, l.workspace_id, l.seed_id, l.branch_id, l.author_id, l.parent_leaf_id,
    l.title, l.summary, l.key_points, l.open_questions, l.tags, l.layer, l.sources, l.metadata, l.created_at, l.position_x, l.position_y, l.search_vector`

// Ancestors returns all ancestor leaves back to the seed root by walking
// the parent_leaf_id chain upward using a recursive CTE.
func (g *GraphQueryEngine) Ancestors(ctx context.Context, leafID types.LeafID) ([]domain.Leaf, error) {
	const op = "exploration: graph ancestors"

	query := fmt.Sprintf(`
		WITH RECURSIVE ancestors AS (
			SELECT %s FROM leaves WHERE id = $1
			UNION ALL
			SELECT %s FROM leaves l
			JOIN ancestors a ON a.parent_leaf_id = l.id
			WHERE a.parent_leaf_id IS NOT NULL
		)
		SELECT %s FROM ancestors WHERE id != $1
		ORDER BY created_at
	`, leafCols, leafColsL, leafCols)

	rows, err := g.db.Query(ctx, query, leafID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}

	leaves, err := scanLeaves(rows)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return leaves, nil
}

// Descendants returns all descendant leaves from a given leaf by walking
// the parent_leaf_id chain downward using a recursive CTE.
func (g *GraphQueryEngine) Descendants(ctx context.Context, leafID types.LeafID) ([]domain.Leaf, error) {
	const op = "exploration: graph descendants"

	query := fmt.Sprintf(`
		WITH RECURSIVE descendants AS (
			SELECT %s FROM leaves WHERE id = $1
			UNION ALL
			SELECT %s FROM leaves l
			JOIN descendants d ON l.parent_leaf_id = d.id
		)
		SELECT %s FROM descendants WHERE id != $1
		ORDER BY created_at
	`, leafCols, leafColsL, leafCols)

	rows, err := g.db.Query(ctx, query, leafID.String())
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}

	leaves, err := scanLeaves(rows)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return leaves, nil
}

// Neighborhood returns all leaves within N hops of a given leaf via
// the connections table. Uses Go-level BFS — each hop queries connections
// containing any frontier leaf, then expands the frontier.
func (g *GraphQueryEngine) Neighborhood(ctx context.Context, leafID types.LeafID, depth int) ([]domain.Leaf, error) {
	const op = "exploration: graph neighborhood"

	if depth <= 0 {
		return nil, nil
	}

	seen := map[string]bool{leafID.String(): true}
	frontier := []string{leafID.String()}

	for hop := 0; hop < depth && len(frontier) > 0; hop++ {
		rows, err := g.db.Query(ctx,
			`SELECT DISTINCT UNNEST(leaf_ids) FROM connections WHERE leaf_ids && $1::text[]`,
			frontier,
		)
		if err != nil {
			return nil, database.MapQueryError(err, op)
		}

		var nextFrontier []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, database.MapQueryError(err, op)
			}
			if !seen[id] {
				seen[id] = true
				nextFrontier = append(nextFrontier, id)
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, database.MapQueryError(err, op)
		}

		frontier = nextFrontier
	}

	// Remove the starting leaf — we only want neighbors.
	delete(seen, leafID.String())

	if len(seen) == 0 {
		return nil, nil
	}

	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}

	return g.fetchLeavesByIDs(ctx, ids, op)
}

// SynthesisLineage traces a synthesis leaf back through its source chain.
// Uses Go-level BFS over the sources JSONB field — each step fetches
// source leaves and recurses into any that are also synthesis leaves.
func (g *GraphQueryEngine) SynthesisLineage(ctx context.Context, leafID types.LeafID) ([]domain.Leaf, error) {
	const op = "exploration: graph synthesis lineage"

	seen := map[string]bool{leafID.String(): true}
	frontier := []string{leafID.String()}
	var result []domain.Leaf

	for len(frontier) > 0 {
		leaves, err := g.fetchLeavesByIDs(ctx, frontier, op)
		if err != nil {
			return nil, err
		}

		var nextFrontier []string
		for _, leaf := range leaves {
			if leaf.ID().String() != leafID.String() {
				result = append(result, leaf)
			}
			for _, src := range leaf.Sources() {
				if !seen[src.LeafID] {
					seen[src.LeafID] = true
					nextFrontier = append(nextFrontier, src.LeafID)
				}
			}
		}

		frontier = nextFrontier
	}

	return result, nil
}

// --- helpers ---

// scanLeaves scans pgx rows into sqlc.Leafe models and converts to domain.
func scanLeaves(rows interface {
	Next() bool
	Scan(dest ...any) error
	Close()
	Err() error
}) ([]domain.Leaf, error) {
	defer rows.Close()

	var models []sqlc.Leafe
	for rows.Next() {
		var m sqlc.Leafe
		if err := rows.Scan(
			&m.ID, &m.WorkspaceID, &m.SeedID, &m.BranchID, &m.AuthorID, &m.ParentLeafID,
			&m.Title, &m.Summary, &m.KeyPoints, &m.OpenQuestions, &m.Tags,
			&m.Layer, &m.Sources, &m.Metadata, &m.CreatedAt, &m.PositionX, &m.PositionY, &m.SearchVector,
		); err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return leavesToDomain(models)
}

// fetchLeavesByIDs batch-fetches leaves by ID using a raw SQL query.
func (g *GraphQueryEngine) fetchLeavesByIDs(ctx context.Context, ids []string, op string) ([]domain.Leaf, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query := fmt.Sprintf(`SELECT %s FROM leaves WHERE id = ANY($1::text[]) ORDER BY created_at`, leafCols)

	rows, err := g.db.Query(ctx, query, ids)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}

	leaves, err := scanLeaves(rows)
	if err != nil {
		return nil, database.MapQueryError(err, op)
	}
	return leaves, nil
}
