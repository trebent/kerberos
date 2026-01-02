package basic

import (
	"net/http"
	"path/filepath"

	"github.com/trebent/kerberos/internal/auth/method"
	"github.com/trebent/kerberos/internal/db"
	"github.com/trebent/kerberos/internal/db/sqlite"
	internalenv "github.com/trebent/kerberos/internal/env"
)

type basic struct {
	db db.SQLClient
}

var _ method.Method = (*basic)(nil)

// New will return an authentication method and register API endpoints with the input serve mux.
func New(mux *http.ServeMux) method.Method {
	a := &basic{
		db: sqlite.New(
			&sqlite.Opts{DSN: filepath.Join(internalenv.DBDirectory.Value(), "auth.db")},
		),
	}
	a.registerAPI(mux)
	return a
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
