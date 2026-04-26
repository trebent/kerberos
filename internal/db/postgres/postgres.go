package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"

	"github.com/lib/pq"
	"github.com/trebent/kerberos/internal/db"
	"github.com/trebent/zerologr"
)

type (
	Opts struct {
		// DSN is a libpq connection string, e.g.
		// "host=localhost port=5432 dbname=kerberos user=krb password=secret sslmode=disable"
		// or a URL: "postgres://krb:secret@localhost:5432/kerberos?sslmode=disable"
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

	namedArgRe = regexp.MustCompile(`@(\w+)`)
)

// New creates a new PostgreSQL client using the provided DSN.
func New(opts *Opts) db.SQLClient {
	sqlDB, err := sql.Open("postgres", opts.DSN)
	if err != nil {
		panic("failed to open postgres connection: invalid driver configuration")
	}

	if err := sqlDB.PingContext(context.Background()); err != nil {
		panic("failed to ping postgres: check DSN and connectivity")
	}

	return &impl{db: sqlDB}
}

// QueryReturningID executes an INSERT ... RETURNING id query and returns the inserted ID.
func QueryReturningID(ctx context.Context, q db.Queryer, query string, args ...any) (int64, error) {
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

func (i *impl) Dialect() db.Dialect {
	return db.PostgresDialect
}

func (i *impl) Begin(ctx context.Context) (db.Transaction, error) {
	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, wrap(err)
	}
	return &txImpl{tx: tx}, nil
}

func (i *impl) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	q, positional := translateNamedArgs(query, args)
	res, err := i.db.ExecContext(ctx, q, positional...)
	if err != nil {
		return nil, wrap(err)
	}
	return res, nil
}

func (i *impl) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	q, positional := translateNamedArgs(query, args)
	rows, err := i.db.QueryContext(ctx, q, positional...)
	if err != nil {
		return nil, wrap(err)
	}
	return rows, nil
}

func (t *txImpl) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	q, positional := translateNamedArgs(query, args)
	res, err := t.tx.ExecContext(ctx, q, positional...)
	if err != nil {
		return nil, wrap(err)
	}
	return res, nil
}

func (t *txImpl) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	q, positional := translateNamedArgs(query, args)
	rows, err := t.tx.QueryContext(ctx, q, positional...)
	if err != nil {
		return nil, wrap(err)
	}
	return rows, nil
}

func (t *txImpl) Commit() error {
	return wrap(t.tx.Commit())
}

func (t *txImpl) Rollback() error {
	if err := t.tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		return wrap(err)
	}
	return nil
}

// translateNamedArgs converts SQLite-style @name parameters to PostgreSQL $N positional parameters.
// Non-NamedArg values are passed through as-is.
func translateNamedArgs(query string, args []any) (string, []any) {
	// Fast path: if no named args, return unchanged.
	hasNamed := false
	for _, a := range args {
		if _, ok := a.(sql.NamedArg); ok {
			hasNamed = true
			break
		}
	}
	if !hasNamed {
		return query, args
	}

	// Build name → value map.
	namedMap := make(map[string]any, len(args))
	for _, a := range args {
		if na, ok := a.(sql.NamedArg); ok {
			namedMap[na.Name] = na.Value
		}
	}

	// Replace @name occurrences in order, assigning $N to each unique name.
	nameToPos := make(map[string]int)
	positional := make([]any, 0, len(args))

	translated := namedArgRe.ReplaceAllStringFunc(query, func(match string) string {
		name := match[1:] // strip leading @
		if pos, exists := nameToPos[name]; exists {
			return fmt.Sprintf("$%d", pos)
		}
		n := len(positional) + 1
		nameToPos[name] = n
		positional = append(positional, namedMap[name])
		return fmt.Sprintf("$%d", n)
	})

	return translated, positional
}

// wrap translates postgres driver errors to Kerberos db errors where appropriate.
func wrap(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" { // unique_violation
			return fmt.Errorf("%w: %w", db.ErrUnique, err)
		}
		zerologr.Info("Unrecognized postgres error", "code", pgErr.Code)
	}

	return err
}
