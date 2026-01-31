package basic

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	apierror "github.com/trebent/kerberos/internal/api/error"
	"github.com/trebent/kerberos/internal/auth/method"
	"github.com/trebent/kerberos/internal/auth/method/basic/api"
	"github.com/trebent/kerberos/internal/db"
	"github.com/trebent/zerologr"
)

type (
	basic struct {
		db db.SQLClient
	}
	Opts struct {
		Mux *http.ServeMux
		DB  db.SQLClient
	}
)

const (
	basicBasePath = "/api/auth/basic"

	queryGetSession = "SELECT user_id, organisation_id, expires FROM sessions WHERE session_id = @sessionID;"
)

var _ method.Method = (*basic)(nil)

// New will return an authentication method and register API endpoints with the input serve mux.
func New(opts *Opts) method.Method {
	b := &basic{
		db: opts.DB,
	}

	b.registerAPI(opts.Mux)
	return b
}

func (a *basic) Authenticated(req *http.Request) error {
	zerologr.V(50).Info("Authenticating request " + req.URL.Path)

	sessionID := req.Header.Get("X-Krb-Session")
	if sessionID == "" {
		zerologr.V(20).Info("Failed to find a session header")

		if zerologr.V(30).Enabled() {
			for key, values := range req.Header {
				zerologr.V(30).Info("Header "+key, "values", values)
			}
		}
		return apierror.APIErrNoSession
	}

	// Read session info from the DB and compare it to the incoming request.
	rows, err := a.db.Query(
		req.Context(),
		queryGetSession,
		sql.NamedArg{Name: "sessionID", Value: sessionID},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query session")
		return apierror.APIErrInternal
	}

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			zerologr.Error(err, "Failed to load next row")
			return apierror.APIErrInternal
		}

		zerologr.Error(err, "Failed to find a matching session")
		return apierror.APIErrNoSession
	}

	var (
		sessionUserID int64
		sessionOrgID  int64
		expires       int64
	)
	err = rows.Scan(&sessionUserID, &sessionOrgID, &expires)
	//nolint:sqlclosecheck // won't help here
	_ = rows.Close()
	if err != nil {
		zerologr.Error(err, "Failed to scan row")
		return apierror.APIErrInternal
	}

	if time.Now().UnixMilli() > expires {
		zerologr.Error(apierror.ErrNoSession, "Session expired")
		return apierror.APIErrNoSession
	}

	req.Header.Set("X-Krb-Org", strconv.Itoa(int(sessionOrgID)))
	req.Header.Set("X-Krb-User", strconv.Itoa(int(sessionUserID)))

	return nil
}

func (a *basic) Authorized(req *http.Request) error {
	zerologr.V(50).Info("Authorizing request " + req.URL.Path)
	return nil
}

func (a *basic) registerAPI(mux *http.ServeMux) {
	ssi := api.NewSSI(a.db)
	_ = api.HandlerFromMuxWithBaseURL(
		api.NewStrictHandlerWithOptions(
			ssi,
			[]api.StrictMiddlewareFunc{api.AuthMiddleware(ssi)},
			api.StrictHTTPServerOptions{
				RequestErrorHandlerFunc:  apierror.RequestErrorHandler,
				ResponseErrorHandlerFunc: apierror.ResponseErrorHandler,
			},
		),
		mux,
		basicBasePath,
	)
}
