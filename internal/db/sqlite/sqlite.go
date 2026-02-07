package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/trebent/kerberos/internal/db"
	"github.com/trebent/zerologr"

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
		return nil, wrap(err)
	}

	_, err = tx.ExecContext(ctx, queryEnableForeignKeys)
	if err != nil {
		return nil, wrap(err)
	}

	return &txImpl{tx: tx}, err
}

func (i *impl) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	res, err := i.db.ExecContext(ctx, query, args...)
	return res, wrap(err)
}

func (i *impl) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	rows, err := i.db.QueryContext(ctx, query, args...)
	return rows, wrap(err)
}

func (t *txImpl) Exec(ctx context.Context, stmt string, args ...any) (sql.Result, error) {
	res, err := t.tx.ExecContext(ctx, stmt, args...)
	return res, wrap(err)
}

func (t *txImpl) Query(ctx context.Context, stmt string, args ...any) (*sql.Rows, error) {
	rows, err := t.tx.QueryContext(ctx, stmt, args...)
	return rows, wrap(err)
}

func (t *txImpl) Commit() error {
	return wrap(t.tx.Commit())
}

func (t *txImpl) Rollback() error {
	return wrap(t.tx.Rollback())
}

func wrap(err error) error {
	if err == nil {
		return err
	}

	code := errorCode(err)
	switch code {
	case 2067:
		return fmt.Errorf("%w: %w", db.ErrUnique, err)
	default:
		zerologr.Info("Unrecognized error code", "code", code)
	}

	return err
}

func errorCode(err error) int {
	var coder interface{ Code() int }
	if errors.As(err, &coder) {
		return coder.Code()
	}

	return 0
}
