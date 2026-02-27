# ── Stage 1: Build ────────────────────────────────────────
FROM golang:1.25-alpine AS build

RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Cache dependency downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /usr/local/bin/canopy-server \
    ./cmd/server/

# Organize migrations by context (avoids filename collisions)
RUN mkdir -p /migrations && \
    for dir in internal/*/adapter/postgres/migrations; do \
        ctx=$(echo "$dir" | cut -d'/' -f2); \
        mkdir -p "/migrations/$ctx"; \
        cp "$dir"/*.sql "/migrations/$ctx/"; \
    done

# ── Stage 2: Runtime ─────────────────────────────────────
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata postgresql16-client \
    && addgroup -S canopy \
    && adduser -S -G canopy canopy

# Copy binary
COPY --from=build /usr/local/bin/canopy-server /usr/local/bin/canopy-server

# Copy organized migrations: /migrations/{context}/001_create_tables.sql
COPY --from=build /migrations /migrations

# Copy migrate script
COPY migrate.sh /migrate.sh
RUN chmod +x /migrate.sh

EXPOSE 8080

USER canopy

ENTRYPOINT ["/usr/local/bin/canopy-server"]
