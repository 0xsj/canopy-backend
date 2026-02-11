# ADR-001: Apache AGE for Graph Queries

## Status
Accepted

## Context
Canopy's exploration context has a graph data model — leaves connect to other leaves, branches form trees, and users need to traverse connections (canvas view, tree view, "show me related ideas"). These queries are naturally graph-shaped: variable-depth traversals, path finding, neighborhood discovery.

The options were:
1. **Neo4j** — established graph database, but adds a second data store, sync pipeline, and operational burden.
2. **DGraph** — Go-native but smaller ecosystem and requires a separate service.
3. **PostgreSQL + Apache AGE** — openCypher graph queries as a Postgres extension, running on the same database.

## Decision
Use Apache AGE as a Postgres extension. Graph data (vertices, edges) lives in the same Postgres instance as relational data. No separate graph database.

## Consequences

**Positive:**
- Single database to operate, back up, and monitor.
- Graph mutations and relational writes in the same transaction — atomic consistency.
- No ETL or sync pipeline between relational and graph stores.
- openCypher query language is well-documented and portable to Neo4j if we ever need to extract.
- Schema-per-context separation still works — the graph is a shared overlay for cross-context traversals.

**Negative:**
- AGE is less mature than Neo4j for advanced graph algorithms (PageRank, community detection).
- Fewer client libraries and tooling compared to Neo4j.
- If graph query volume becomes very high, we can't scale the graph independently of Postgres.
- Team needs to learn openCypher syntax alongside SQL.
