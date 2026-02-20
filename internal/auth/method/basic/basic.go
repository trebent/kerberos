package basic

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path"
	"slices"
	"strconv"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	api "github.com/trebent/kerberos/internal/api/auth/basic"
	apierror "github.com/trebent/kerberos/internal/api/error"
	"github.com/trebent/kerberos/internal/auth/method"
	basicapi "github.com/trebent/kerberos/internal/auth/method/basic/api"
	composertypes "github.com/trebent/kerberos/internal/composer/types"

	"github.com/trebent/kerberos/internal/db"
	"github.com/trebent/kerberos/internal/oas"
	"github.com/trebent/zerologr"
)

type (
	basic struct {
		db     db.SQLClient
		oasDir string
		config map[string]method.AuthZConfig
	}
	Opts struct {
		AuthZConfig map[string]method.AuthZConfig

		Mux                    *http.ServeMux
		DB                     db.SQLClient
		OASDir                 string
		AdminSessionMiddleware api.StrictMiddlewareFunc
	}
)

const (
	authBasicSpecification = "auth_basic.yaml"

	queryGetSession     = "SELECT user_id, organisation_id, expires FROM sessions WHERE session_id = @sessionID;"
	queryListUserGroups = "SELECT name FROM groups WHERE id IN (SELECT group_id FROM group_bindings WHERE user_id = @userID) AND organisation_id = @orgID;"
)

var _ method.Method = (*basic)(nil)

// New will return an authentication method and register API endpoints with the input serve mux.
func New(opts *Opts) method.Method {
	b := &basic{
		db:     opts.DB,
		oasDir: opts.OASDir,
		config: opts.AuthZConfig,
	}

	b.registerAPI(opts)
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

//nolint:gocognit // welp
func (a *basic) Authorized(req *http.Request) error {
	zerologr.V(50).Info("Authorizing request " + req.URL.Path)
	//nolint:errcheck // bigger problems if this is missing
	backend := req.Context().Value(composertypes.BackendContextKey).(string)

	authZ, ok := a.config[backend]
	if !ok {
		zerologr.V(50).Info("No authorization scheme defined for backend " + backend)
		return nil
	}

	var groupsToValidate []string
	// Check if path override is present for the backend:
	for p, pathGroups := range authZ.Paths {
		match, err := path.Match(p, req.URL.Path)
		if err != nil {
			return err
		}

		if match {
			groupsToValidate = pathGroups
			break
		}
	}

	// Return nil if neither global groups are configured, nor any path override exists.
	if len(groupsToValidate) == 0 && len(authZ.Groups) == 0 {
		zerologr.V(50).Info("No authorization group mapping defined for backend " + backend)
		return nil
	}

	// Set validation groups depending on if global or path override.
	if len(groupsToValidate) == 0 {
		zerologr.V(50).Info("Validating global group mapping for " + backend)
		groupsToValidate = authZ.Groups
	} else {
		zerologr.V(50).Info("Validating path group mapping for " + backend)
	}

	// Fetch the user's groups.
	rows, err := a.db.Query(
		req.Context(),
		queryListUserGroups,
		sql.NamedArg{Name: "orgID", Value: req.Header.Get("X-Krb-Org")},
		sql.NamedArg{Name: "userID", Value: req.Header.Get("X-Krb-User")},
	)
	if err != nil {
		zerologr.Error(err, "Failed to query user groups")
		return apierror.APIErrInternal
	}
	// Fine to defer since we're iterating, not just doing one scan.
	defer rows.Close()

	userGroups := make([]string, 0)
	for {
		if !rows.Next() {
			if err := rows.Err(); err != nil {
				zerologr.Error(err, "Failed to scan next row")
				return apierror.APIErrInternal
			}

			break
		}

		groupName := ""
		if err = rows.Scan(&groupName); err != nil {
			zerologr.Error(err, "Failed to scan row")
			return apierror.APIErrInternal
		}

		userGroups = append(userGroups, groupName)
		req.Header.Add("X-Krb-Groups", groupName)
	}

	for _, usergroup := range userGroups {
		if slices.Contains(groupsToValidate, usergroup) {
			return nil
		}
	}

	// No group match found -> 403
	return apierror.APIErrForbidden
}

func (a *basic) registerAPI(opts *Opts) {
	data, err := os.ReadFile(fmt.Sprintf("%s/%s", a.oasDir, authBasicSpecification))
	if err != nil {
		panic(fmt.Errorf("failed to read basic authentication OAS: %w", err))
	}

	spec, err := openapi3.NewLoader().LoadFromData(data)
	if err != nil {
		panic(fmt.Errorf("failed to load basic authentication OAS: %w", err))
	}

	ssi := basicapi.NewSSI(a.db)
	strictHandler := api.NewStrictHandlerWithOptions(
		ssi,
		[]api.StrictMiddlewareFunc{
			basicapi.AuthMiddleware(ssi),
			opts.AdminSessionMiddleware,
		},
		api.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  apierror.RequestErrorHandler,
			ResponseErrorHandlerFunc: apierror.ResponseErrorHandler,
		},
	)

	_ = api.HandlerWithOptions(strictHandler, api.StdHTTPServerOptions{
		BaseRouter: opts.Mux,
		Middlewares: []api.MiddlewareFunc{
			oas.ValidationMiddleware(spec),
			func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					zerologr.Info(fmt.Sprintf("%s %s", r.Method, r.URL.Path))
					next.ServeHTTP(w, r)
				})
			},
		},
	})
}
