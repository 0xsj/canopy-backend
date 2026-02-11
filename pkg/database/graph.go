package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Graph provides helpers for executing Apache AGE Cypher queries
// on the canopy graph. Wraps the AGE-specific SQL boilerplate so
// domain code only writes Cypher.
type Graph struct {
	db        *DB
	graphName string
}

// NewGraph creates a graph query helper for the named graph.
func NewGraph(db *DB, graphName string) *Graph {
	return &Graph{db: db, graphName: graphName}
}

// Query executes a Cypher query and returns rows.
// The caller must define the result columns and types in the AS clause
// via the columns parameter.
//
// Acquires a single connection from the pool so that LOAD 'age',
// SET search_path, and the Cypher query all run on the same connection.
//
// Example:
//
//	rows, err := graph.Query(ctx,
//	    "MATCH (l:Leaf {id: $id})-[:CONNECTED_TO]-(r:Leaf) RETURN r",
//	    "(leaf agtype)",
//	    pgx.NamedArgs{"id": leafID},
//	)
func (g *Graph) Query(ctx context.Context, cypher string, columns string, args ...any) (pgx.Rows, error) {
	conn, err := g.db.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("database: graph acquire conn: %w", err)
	}
	// Note: caller must close rows, which returns the connection to the pool.
	// We release on error only.

	if err := g.ensureAGE(ctx, conn); err != nil {
		conn.Release()
		return nil, err
	}

	sql := fmt.Sprintf(
		`SELECT * FROM cypher('%s', $$ %s $$) AS %s`,
		g.graphName, cypher, columns,
	)

	rows, err := conn.Query(ctx, sql, args...)
	if err != nil {
		conn.Release()
		return nil, fmt.Errorf("database: graph query: %w", err)
	}

	return rows, nil
}

// Exec executes a Cypher mutation (CREATE, SET, DELETE) that returns no rows.
//
// Example:
//
//	err := graph.Exec(ctx,
//	    "CREATE (l:Leaf {id: $id, title: $title})",
//	    pgx.NamedArgs{"id": leafID, "title": title},
//	)
func (g *Graph) Exec(ctx context.Context, cypher string, args ...any) error {
	conn, err := g.db.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("database: graph acquire conn: %w", err)
	}
	defer conn.Release()

	if err := g.ensureAGE(ctx, conn); err != nil {
		return err
	}

	sql := fmt.Sprintf(
		`SELECT * FROM cypher('%s', $$ %s $$) AS (result agtype)`,
		g.graphName, cypher,
	)

	_, err = conn.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("database: graph exec: %w", err)
	}
	return nil
}

// ensureAGE loads the AGE extension and sets the search path on a
// specific acquired connection. Must be called per-connection since
// LOAD and SET search_path are connection-scoped.
func (g *Graph) ensureAGE(ctx context.Context, conn *pgxpool.Conn) error {
	_, err := conn.Exec(ctx, "LOAD 'age'")
	if err != nil {
		return fmt.Errorf("database: load age: %w", err)
	}

	_, err = conn.Exec(ctx, `SET search_path = ag_catalog, "$user", public`)
	if err != nil {
		return fmt.Errorf("database: set age search_path: %w", err)
	}

	return nil
}
