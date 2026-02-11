package database

import (
	"context"
	"fmt"
	"time"
)

// DefaultHealthTimeout is the default timeout for health checks.
const DefaultHealthTimeout = 3 * time.Second

// Health checks that the database is reachable and responsive.
// Returns nil if healthy, an error describing the issue otherwise.
func (db *DB) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultHealthTimeout)
	defer cancel()

	if err := db.pool.Ping(ctx); err != nil {
		return fmt.Errorf("database: health check failed: %w", err)
	}
	return nil
}

// Stats returns current connection pool statistics.
type Stats struct {
	TotalConns   int32
	IdleConns    int32
	AcquiredConn int32
}

// Stats returns the current pool statistics.
func (db *DB) Stats() Stats {
	s := db.pool.Stat()
	return Stats{
		TotalConns:   s.TotalConns(),
		IdleConns:    s.IdleConns(),
		AcquiredConn: s.AcquiredConns(),
	}
}
