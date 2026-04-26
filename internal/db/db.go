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

// QueryReturningID executes an INSERT ... RETURNING id query and returns the inserted ID.
func QueryReturningID(ctx context.Context, q Queryer, query string, args ...any) (int64, error) {
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, err
		}
		return 0, errors.New("insert returned no id")
	}

	var id int64
	if err := rows.Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
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
