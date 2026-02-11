package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// WithTx runs fn inside a database transaction. If fn returns nil,
// the transaction is committed. If fn returns an error or panics,
// the transaction is rolled back.
//
// The DBTX passed to fn is the transaction itself — pass it to your
// sqlc-generated querier to run queries within the transaction.
func (db *DB) WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("database: begin transaction: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback(ctx)
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("database: rollback failed: %w (original: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("database: commit: %w", err)
	}

	return nil
}

// WithTxOptions runs fn inside a transaction with custom isolation/access options.
func (db *DB) WithTxOptions(ctx context.Context, opts pgx.TxOptions, fn func(tx pgx.Tx) error) error {
	tx, err := db.pool.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("database: begin transaction: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback(ctx)
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("database: rollback failed: %w (original: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("database: commit: %w", err)
	}

	return nil
}
