package sqlite

import (
	"context"
	"database/sql"
	"errors"

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
	}
	txImpl struct {
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
	db.SetMaxIdleConns(0)

	if err := db.PingContext(context.Background()); err != nil {
		panic(err)
	}

	return &impl{db: db}
}

func (i *impl) Begin(ctx context.Context) (db.Transaction, error) {
	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx, queryEnableForeignKeys)
	if err != nil {
		return nil, err
	}

	return &txImpl{tx: tx}, err
}

func (i *impl) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return i.db.ExecContext(ctx, query, args...)
}

func (i *impl) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return i.db.QueryContext(ctx, query, args...)
}

func (t *txImpl) Exec(stmt string, args ...any) (sql.Result, error) {
	return t.tx.Exec(stmt, args...)
}

func (t *txImpl) Query(stmt string, args ...any) (*sql.Rows, error) {
	return t.tx.Query(stmt, args...)
}

func (t *txImpl) Commit() error {
	return t.tx.Commit()
}

func (t *txImpl) Rollback() error {
	return t.tx.Rollback()
}

func (t *impl) ErrorCode(err error) int {
	var coder interface{ Code() int }
	if errors.As(err, &coder) {
		return coder.Code()
	}

	return 0
}
