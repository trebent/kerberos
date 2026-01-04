package basic

import (
	"context"
	_ "embed"
	"net/http"
	"path/filepath"
	"time"

	"github.com/trebent/kerberos/internal/auth/method"
	"github.com/trebent/kerberos/internal/auth/method/basic/api"
	"github.com/trebent/kerberos/internal/db"
	"github.com/trebent/kerberos/internal/db/sqlite"
	internalenv "github.com/trebent/kerberos/internal/env"
)

type basic struct {
	db db.SQLClient
}

var (
	_ method.Method = (*basic)(nil)

	//go:embed dbschema/schema.sql
	schemaBytes []byte
)

const schemaApplyTimeout = 10 * time.Second

// New will return an authentication method and register API endpoints with the input serve mux.
func New(mux *http.ServeMux) method.Method {
	b := &basic{
		db: sqlite.New(
			&sqlite.Opts{DSN: filepath.Join(internalenv.DBDirectory.Value(), "auth.db")},
		),
	}

	b.applySchemas()
	b.registerAPI(mux)
	return b
}

func (a *basic) Authenticated(*http.Request) error {
	// Read session info from the DB and compare it to the incoming request.
	return nil
}

func (a *basic) Authorized(*http.Request) error {
	return nil
}

func (a *basic) registerAPI(mux *http.ServeMux) {
	_ = api.HandlerFromMuxWithBaseURL(
		api.NewStrictHandler(api.NewSSI(a.db), []api.StrictMiddlewareFunc{api.AuthMiddleware}),
		mux,
		"/api/auth/basic",
	)
}

func (a *basic) applySchemas() {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), schemaApplyTimeout)
	defer cancel()
	if _, err := a.db.Exec(timeoutCtx, string(schemaBytes)); err != nil {
		panic(err)
	}
}
