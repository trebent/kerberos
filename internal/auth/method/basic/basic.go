package basic

import (
	"context"
	_ "embed"
	"net/http"
	"path/filepath"
	"time"

	"github.com/trebent/kerberos/internal/auth/method"
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
	_ = HandlerFromMuxWithBaseURL(
		NewStrictHandler(NewSSI(a.db), []StrictMiddlewareFunc{}),
		mux,
		"/api/auth/basic",
	)
}

func (a *basic) applySchemas() {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := a.db.ApplySchema(timeoutCtx, schemaBytes); err != nil {
		panic(err)
	}
}
