package sqlite

import (
	"context"
	"database/sql"

	"github.com/trebent/kerberos/internal/db"
	// this is how it works.
	_ "modernc.org/sqlite"
)

type (
	Opts struct {
		DSN string
	}
	impl struct {
		conn *sql.DB
	}
)

var _ db.SQLClient = (*impl)(nil)

const driver = "sqlite"

func New(opts *Opts) db.SQLClient {
	conn, err := sql.Open(driver, opts.DSN)
	if err != nil {
		panic(err)
	}

	if err := conn.PingContext(context.Background()); err != nil {
		panic(err)
	}

	return &impl{conn}
}

func (i *impl) Select() error {
	return nil
}
