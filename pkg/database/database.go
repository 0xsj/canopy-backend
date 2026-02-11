package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/0xsj/canopy-backend/pkg/config"
	"github.com/0xsj/canopy-backend/pkg/observability/logger"
)

// DBTX is the interface satisfied by both *pgxpool.Pool and pgx.Tx.
// sqlc-generated code depends on this shape — adapters accept DBTX so
// they work transparently inside or outside a transaction.
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// DB wraps a pgx connection pool and provides the entry point for
// all database operations. Passed via constructor injection.
type DB struct {
	pool *pgxpool.Pool
	log  logger.Logger
}

// Connect creates a new database connection pool from the given config.
// Fails fast if the connection cannot be established.
func Connect(ctx context.Context, cfg Config, log logger.Logger) (*DB, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("database: parse config: %w", err)
	}

	poolCfg.MaxConns = int32(cfg.MaxPoolSize)
	poolCfg.MinConns = int32(cfg.MinPoolSize)
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("database: connect: %w", err)
	}

	// Verify connection immediately — fail fast.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database: ping: %w", err)
	}

	log.Info("database connected",
		logger.String("dsn", cfg.sanitizedDSN()),
		logger.Int("max_pool", cfg.MaxPoolSize),
	)

	return &DB{pool: pool, log: log}, nil
}

// Pool returns the underlying pgxpool.Pool.
// Use this when you need pool-level access (e.g., Acquire).
func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

// DBTX returns the pool as a DBTX interface.
// Adapters accept DBTX so they work with both pool and transaction.
func (db *DB) DBTX() DBTX {
	return db.pool
}

// Close shuts down the connection pool.
func (db *DB) Close() {
	db.pool.Close()
	db.log.Info("database connection closed")
}

// Config holds database connection settings.
// Implements config.Section for use with the config loader.
type Config struct {
	DSN             string
	MaxPoolSize     int
	MinPoolSize     int
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

func (c *Config) Load(env config.EnvReader) {
	c.DSN = env.String("DSN", "postgres://canopy:canopy@localhost:5433/canopy?sslmode=disable")
	c.MaxPoolSize = env.Int("MAX_POOL_SIZE", 20)
	c.MinPoolSize = env.Int("MIN_POOL_SIZE", 5)
	c.MaxConnLifetime = env.Duration("MAX_CONN_LIFETIME", 30*time.Minute)
	c.MaxConnIdleTime = env.Duration("MAX_CONN_IDLE_TIME", 5*time.Minute)
}

func (c *Config) Validate() error {
	var v config.Errors
	v.Required("dsn", c.DSN)
	v.Positive("max_pool_size", c.MaxPoolSize)
	v.Positive("min_pool_size", c.MinPoolSize)
	return v.Err()
}

// sanitizedDSN returns the DSN with the password masked for logging.
func (c *Config) sanitizedDSN() string {
	cfg, err := pgx.ParseConfig(c.DSN)
	if err != nil {
		return "<invalid>"
	}
	return fmt.Sprintf("%s@%s:%d/%s", cfg.User, cfg.Host, cfg.Port, cfg.Database)
}
