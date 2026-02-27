#!/bin/sh
set -e

# Migration order matches bounded context dependencies.
CONTEXTS="identity organization workspace seed exploration discussion convergence session synthesis deliverable notification ledger"

DSN="${CANOPY_DATABASE_DSN:?CANOPY_DATABASE_DSN is required}"

echo "running migrations..."

for ctx in $CONTEXTS; do
    dir="/migrations/$ctx"
    if [ -d "$dir" ]; then
        for sql in "$dir"/*.sql; do
            echo "  migrate: $ctx/$(basename "$sql")"
            psql "$DSN" -f "$sql" -q 2>&1 | grep -v "already exists" || true
        done
    fi
done

echo "migrations complete"
