package db

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Queryer is satisfied by both SQLClient and Transaction.
type Queryer interface {
	Query(ctx context.Context, stmt string, args ...any) (*sql.Rows, error)
}

// Dialect identifies the underlying database engine.
type Dialect string

const (
	SQLiteDialect   Dialect = "sqlite"
	PostgresDialect Dialect = "postgres"
)

type (
	SQLClient interface {
		// Begin starts a new transaction.
		Begin(ctx context.Context) (Transaction, error)
		// Exec executes a standard query without returning any rows.
		Exec(ctx context.Context, stmt string, args ...any) (sql.Result, error)
		// Query executes a standard query, returning resulting rows.
		Query(ctx context.Context, stmt string, args ...any) (*sql.Rows, error)
		// Dialect returns the underlying database dialect.
		Dialect() Dialect
	}
	Transaction interface {
		// Exec executes a standard query without returning any rows.
		Exec(ctx context.Context, stmt string, args ...any) (sql.Result, error)
		// Query executes a standard query, returning resulting rows.
		Query(ctx context.Context, stmt string, args ...any) (*sql.Rows, error)
		// Commit commits the transaction.
		Commit() error
		// Rollback rolls the transaction back.
		Rollback() error
	}
)

// Failed unique constraint, conflict.
var ErrUnique = errors.New("unique constraint failed")

const SchemaApplyTimeout = 5 * time.Second
