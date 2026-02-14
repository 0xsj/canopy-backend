CONTEXTS := identity organization workspace seed exploration discussion \
            convergence session synthesis deliverable notification ledger

INFRA_DIR   := ../canopy-infra
DB_DSN      := postgres://canopy:canopy@localhost:5433/canopy?sslmode=disable

# Load .env.dev if it exists (for LLM keys, overrides, etc.)
-include .env.dev
export

# ── Development ─────────────────────────────────────────────

.PHONY: dev server build

## dev: start infra, apply migrations, run server
dev: infra-up migrate server

## server: run the canopy server with dev defaults
server:
	go run ./cmd/server/

## build: compile the server binary
build:
	go build -o bin/canopy-server ./cmd/server/

# ── Infrastructure ──────────────────────────────────────────

.PHONY: infra-up infra-down infra-status infra-logs

## infra-up: start postgres, nats, redis via docker-compose
infra-up:
	docker compose -f $(INFRA_DIR)/docker-compose.yml --env-file $(INFRA_DIR)/.env up -d
	@echo "waiting for containers to be healthy..."
	@until docker inspect canopy-postgres --format='{{.State.Health.Status}}' 2>/dev/null | grep -q healthy; do sleep 1; done
	@until docker inspect canopy-nats --format='{{.State.Health.Status}}' 2>/dev/null | grep -q healthy; do sleep 1; done
	@echo "infra ready: postgres=5433 nats=4223 redis=6380"

## infra-down: stop and remove containers
infra-down:
	docker compose -f $(INFRA_DIR)/docker-compose.yml --env-file $(INFRA_DIR)/.env down

## infra-status: show container status
infra-status:
	@docker ps --filter name=canopy --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'

## infra-logs: tail logs from all infra containers
infra-logs:
	docker compose -f $(INFRA_DIR)/docker-compose.yml --env-file $(INFRA_DIR)/.env logs -f

# ── Database ────────────────────────────────────────────────

.PHONY: migrate migrate-status db-clean db-reset db-psql

## migrate: apply all context migrations in order
migrate:
	@for ctx in $(CONTEXTS); do \
		migration="internal/$$ctx/adapter/postgres/migrations/001_create_tables.sql"; \
		if [ -f "$$migration" ]; then \
			echo "migrate: $$ctx"; \
			psql "$(DB_DSN)" -f "$$migration" -q 2>&1 | grep -v "already exists" || true; \
		fi; \
	done
	@echo "migrations complete"

## migrate-status: show all tables across context schemas
migrate-status:
	@psql "$(DB_DSN)" -c "\dt identity.* organization.* workspace.* seed.* exploration.* discussion.* convergence.* session.* synthesis.* deliverable.* notification.* ledger.*"

## db-clean: truncate all data tables (keeps schemas and migrations)
db-clean:
	@echo "Truncating all data tables..."
	@psql "$(DB_DSN)" -q -c " \
		DO \$$\$$ \
		DECLARE t record; \
		BEGIN \
			FOR t IN \
				SELECT schemaname, tablename FROM pg_tables \
				WHERE schemaname IN ('identity','organization','workspace','seed','exploration', \
					'discussion','convergence','session','synthesis','deliverable','notification','ledger','canopy') \
				AND tablename != 'schema_migrations' \
			LOOP \
				EXECUTE format('TRUNCATE %I.%I CASCADE', t.schemaname, t.tablename); \
			END LOOP; \
		END \$$\$$;"
	@echo "all data cleared"

## db-reset: drop and recreate all context schemas (destructive!)
db-reset:
	@echo "⚠ This will drop ALL context schemas. Press Ctrl+C to cancel."
	@read -p "Type 'yes' to confirm: " confirm && [ "$$confirm" = "yes" ] || exit 1
	@for ctx in $(CONTEXTS); do \
		echo "dropping schema: $$ctx"; \
		psql "$(DB_DSN)" -c "DROP SCHEMA IF EXISTS $$ctx CASCADE;" -q; \
	done
	psql "$(DB_DSN)" -f $(INFRA_DIR)/postgres/init.sql -q
	@$(MAKE) migrate
	@echo "database reset complete"

## db-psql: open interactive psql session
db-psql:
	psql "$(DB_DSN)"

# ── Code Quality ────────────────────────────────────────────

.PHONY: fmt test test-verbose test-coverage test-unit sqlc-generate lint

## fmt: format all Go files
fmt:
	go fmt ./...

## test: run all tests
test:
	go test ./...

## test-verbose: run all tests with verbose output
test-verbose:
	go test -v ./...

## test-unit: run only unit tests (skip integration)
test-unit:
	go test -short ./...

## test-coverage: run tests with coverage report
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

## sqlc-generate: regenerate sqlc code for all contexts
sqlc-generate:
	@for ctx in $(CONTEXTS); do \
		echo "sqlc generate: $$ctx"; \
		pushd internal/$$ctx/adapter/postgres > /dev/null && sqlc generate && popd > /dev/null; \
	done

# ── Smoke Test ──────────────────────────────────────────────

.PHONY: smoke-test

BASE_URL := http://localhost:8080

## smoke-test: exercise the full API flow (server must be running)
smoke-test:
	@echo "=== Canopy Smoke Test ==="
	@echo ""
	@echo "--- Health ---"
	curl -sf $(BASE_URL)/healthz/live | jq .
	curl -sf $(BASE_URL)/healthz/ready | jq .
	@echo ""
	@echo "--- Identity: create user ---"
	$(eval USER := $(shell curl -sf -X POST $(BASE_URL)/api/v1/users \
		-H 'Content-Type: application/json' \
		-d '{"display_name":"Smoke Tester","email":"smoke@canopy.dev","auth_provider":"local","auth_provider_id":"smoke_001"}' | tee /dev/stderr | jq -r '.id'))
	@echo "user_id: $(USER)"
	@echo ""
	@echo "--- Organization: create org ---"
	$(eval ORG := $(shell curl -sf -X POST $(BASE_URL)/api/v1/orgs \
		-H 'Content-Type: application/json' \
		-d '{"name":"Smoke Org","slug":"smoke-org"}' | tee /dev/stderr | jq -r '.id'))
	@echo "org_id: $(ORG)"
	@echo ""
	@echo "--- Workspace: create workspace ---"
	$(eval WS := $(shell curl -sf -X POST $(BASE_URL)/api/v1/orgs/$(ORG)/workspaces \
		-H 'Content-Type: application/json' \
		-d '{"name":"Smoke Workspace","description":"Testing the full flow"}' | tee /dev/stderr | jq -r '.id'))
	@echo "workspace_id: $(WS)"
	@echo ""
	@echo "--- Seed: plant seed ---"
	$(eval SEED := $(shell curl -sf -X POST $(BASE_URL)/api/v1/workspaces/$(WS)/seeds \
		-H 'Content-Type: application/json' \
		-d '{"title":"Distributed Consensus","description":"Exploring consensus algorithms in distributed systems"}' | tee /dev/stderr | jq -r '.id'))
	@echo "seed_id: $(SEED)"
	@echo ""
	@echo "--- Exploration: start branch (creates branch + root leaf) ---"
	$(eval BRANCH_RESP := $(shell curl -sf -X POST $(BASE_URL)/api/v1/workspaces/$(WS)/branches \
		-H 'Content-Type: application/json' \
		-d '{"seed_id":"$(SEED)","title":"Raft Consensus","summary":"Raft is designed for understandability","key_points":["Strong leader model","Log replication"],"open_questions":["How does it compare to Paxos?"],"tags":["raft","consensus"]}' | tee /dev/stderr))
	$(eval LEAF1 := $(shell echo '$(BRANCH_RESP)' | jq -r '.leaf.id'))
	@echo "leaf_1_id: $(LEAF1)"
	@echo ""
	@echo "--- Exploration: create second leaf ---"
	$(eval LEAF2 := $(shell curl -sf -X POST $(BASE_URL)/api/v1/workspaces/$(WS)/leaves \
		-H 'Content-Type: application/json' \
		-d '{"seed_id":"$(SEED)","branch_id":"$(shell echo '$(BRANCH_RESP)' | jq -r '.branch.id')","title":"Paxos Protocol","summary":"Paxos optimizes for correctness proofs","key_points":["Flexible quorum model","Harder to implement"],"open_questions":["What about Byzantine faults?"],"tags":["paxos","consensus"]}' | tee /dev/stderr | jq -r '.id'))
	@echo "leaf_2_id: $(LEAF2)"
	@echo ""
	@echo "--- Synthesis: merge two leaves ---"
	curl -sf -X POST $(BASE_URL)/api/v1/workspaces/$(WS)/syntheses \
		-H 'Content-Type: application/json' \
		-d '{"source_leaf_ids":["$(LEAF1)","$(LEAF2)"]}' | jq .
	@echo ""
	@echo "=== Smoke test complete ==="

# ── Help ────────────────────────────────────────────────────

.PHONY: help

## help: show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /' | sort
