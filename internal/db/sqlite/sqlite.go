package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/trebent/kerberos/internal/db"
	// this is how it works.
	_ "modernc.org/sqlite"
)

type (
	Opts struct {
		DSN string
	}
	// PRAGMA foreign_keys=ON; needs to be run per connection to enforce FKs.
	impl struct {
		db *sql.DB
		tx *sql.Tx
	}
)

var (
	_ db.SQLClient = (*impl)(nil)

	ErrNoTransaction = errors.New("no transaction")
)

const (
	DBName = "krb.db"

	driver = "sqlite"

	queryEnableForeignKeys = "PRAGMA foreign_keys=ON;"
)

func New(opts *Opts) db.SQLClient {
	db, err := sql.Open(driver, opts.DSN)
	if err != nil {
		panic(err)
	}

	// Sqlite does not like concurrency.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.PingContext(context.Background()); err != nil {
		panic(err)
	}

	return &impl{db: db}
}

func (i *impl) Begin(ctx context.Context) (db.SQLClient, error) {
	ni := *i
	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx, queryEnableForeignKeys)
	if err != nil {
		return nil, err
	}

	ni.tx = tx
	return &ni, err
}

func (i *impl) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if i.tx != nil {
		return i.tx.ExecContext(ctx, query, args...)
	}

	return i.db.ExecContext(ctx, query, args...)
}

func (i *impl) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if i.tx != nil {
		return i.tx.QueryContext(ctx, query, args...)
	}

	return i.db.QueryContext(ctx, query, args...)
}

func (i *impl) Commit() error {
	if i.tx != nil {
		return i.tx.Commit()
	}

	return fmt.Errorf("%w: cannot commit", ErrNoTransaction)
}

func (i *impl) Rollback() error {
	if i.tx != nil {
		return i.tx.Rollback()
	}

	return fmt.Errorf("%w: cannot rollback", ErrNoTransaction)
}
