package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/trebent/kerberos/internal/db"
	"github.com/trebent/zerologr"

	// this is how it works.
	_ "modernc.org/sqlite"
)

type (
	Opts struct {
		DSN string
	}
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
)

// New creates a new SQLite client. If opts.DSN is empty, it defaults to "krb.db" in the current directory.
// The input DNS is modified to ensure that foreign key constraints are enabled.
func New(opts *Opts) db.SQLClient {
	dsn := opts.DSN
	if dsn == "" {
		dsn = DBName
	}

	if strings.Contains(dsn, "?") {
		dsn += "&_pragma=foreign_keys=on"
	} else {
		dsn += "?_pragma=foreign_keys=on"
	}

	db, err := sql.Open(driver, dsn)
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

	return &txImpl{tx: tx}, nil
}

func (i *impl) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	res, err := i.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, wrap(err)
	}
	return res, nil
}

func (i *impl) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	rows, err := i.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, wrap(err)
	}
	return rows, nil
}

func (t *txImpl) Exec(ctx context.Context, stmt string, args ...any) (sql.Result, error) {
	res, err := t.tx.ExecContext(ctx, stmt, args...)
	if err != nil {
		return nil, wrap(err)
	}
	return res, nil
}

func (t *txImpl) Query(ctx context.Context, stmt string, args ...any) (*sql.Rows, error) {
	rows, err := t.tx.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, wrap(err)
	}
	return rows, nil
}

func (t *txImpl) Commit() error {
	if err := t.tx.Commit(); err != nil {
		return wrap(err)
	}
	return nil
}

func (t *txImpl) Rollback() error {
	if err := t.tx.Rollback(); err != nil {
		return wrap(err)
	}
	return nil
}

func wrap(err error) error {
	if err == nil {
		return nil
	}

	code := errorCode(err)
	switch code {
	case 1555: // primary key duplication/violation
		return fmt.Errorf("%w: %w", db.ErrUnique, err)
	case 2067:
		return fmt.Errorf("%w: %w", db.ErrUnique, err)
		// SQLITE OK
	case 0:
		return nil
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
