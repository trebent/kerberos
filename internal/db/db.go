package db

import (
	"context"
	"database/sql"
)

type SQLClient interface {
	Begin(ctx context.Context) (SQLClient, error)
	Exec(ctx context.Context, stmt string, args ...any) (sql.Result, error)
	Query(ctx context.Context, stmt string, args ...any) (*sql.Rows, error)
	Commit() error
	Rollback() error
}
