# 2026-02-11 — Infrastructure Setup

## What was done
- Built `canopy-infra/docker-compose.yml` — three services: Postgres with Apache AGE, NATS with JetStream, Redis
- Postgres uses `apache/age:release_PG17_1.6.0` image with health checks via `pg_isready`, data persisted to a named volume, and init script mounted at `/docker-entrypoint-initdb.d/`
- Built `canopy-infra/postgres/init.sql` — enables the AGE extension, creates the `canopy` graph, and creates one schema per bounded context: identity, workspace, seed, exploration, synthesis, session, convergence, discussion, deliverable, notification
- Built `canopy-infra/nats/nats-server.conf` — enables JetStream with 256MB memory store and 1GB file store, exposes client port 4222 and monitoring port 8222
- Redis runs `redis:7-alpine` with health checks via `redis-cli ping` and data persisted to a named volume
- Built `canopy-infra/.env` — default credentials and port mappings (Postgres on 5433, NATS on 4222/8222, Redis on 6379)
- All three services have health checks with 5s interval, 3s timeout, 5 retries
- Wrote ADR-001: Apache AGE for Graph Queries — documenting the decision to use AGE over Neo4j or DGraph

## Decisions made
- Apache AGE over a separate graph database — single Postgres instance handles both relational and graph data, no ETL pipeline, atomic transactions across both models (see ADR-001)
- One Postgres schema per bounded context — enforces schema separation at the database level, no cross-schema joins, matches the architecture doc's boundary rules
- Postgres exposed on port 5433 (not default 5432) — avoids conflicts with a local Postgres installation
- JetStream enabled by default — NATS provides at-least-once delivery and durable subscriptions for domain events out of the box
- Redis included from the start — ready for session cache and ephemeral state even though no adapter uses it yet
- Named Docker volumes for all three services — data survives container recreation

## Open questions
- Should the init.sql also create the `schema_migrations` tables per schema, or leave that to the Migrator at runtime?
- Redis is provisioned but has no adapter or configuration wired yet — when does that get built?
- NATS JetStream limits (256MB mem, 1GB file) are development defaults — what are production targets?

## Next steps
- Build the `pkg/database` package to connect to Postgres from Go
- Build the `pkg/nats` package to connect to NATS and manage JetStream streams
- Wire infrastructure into the composition root (`cmd/server/`)
