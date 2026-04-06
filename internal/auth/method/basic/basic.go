package basic

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"slices"
	"strconv"
	"time"

	_ "embed"

	"github.com/getkin/kin-openapi/openapi3"
	adminext "github.com/trebent/kerberos/internal/admin/extensions"
	"github.com/trebent/kerberos/internal/auth/method"
	"github.com/trebent/kerberos/internal/composer"
	"github.com/trebent/kerberos/internal/config"
	authbasicapi "github.com/trebent/kerberos/internal/oapi/auth/basic"
	apierror "github.com/trebent/kerberos/internal/oapi/error"

	"github.com/trebent/kerberos/internal/db"
	"github.com/trebent/kerberos/internal/oas"
	"github.com/trebent/zerologr"
)

type (
	Basic interface {
		method.Method
		adminext.APIProvider
	}
	basic struct {
		config    map[string]*config.AuthZ
		sqlClient db.SQLClient
		oasDir    string
	}
	Opts struct {
		AuthZConfig map[string]*config.AuthZ
		SQLClient   db.SQLClient
		OASDir      string
	}
)

const authBasicSpecification = "auth_basic.yaml"

var (
	_ Basic = (*basic)(nil)

	//go:embed dbschema/schema.sql
	dbschemaBytes []byte
)

// New will return an authentication method and register API endpoints with the input serve mux.
func New(opts *Opts) (Basic, error) {
	if opts.SQLClient == nil {
		return nil, errors.New("DB client is required for basic auth method")
	}

	if err := applySchemas(opts.SQLClient); err != nil {
		return nil, fmt.Errorf("failed to apply basic auth DB schema: %w", err)
	}

	if opts.OASDir == "" {
		return nil, errors.New("OAS directory is required for basic auth method")
	}

	if opts.AuthZConfig == nil {
		return nil, errors.New("authorization config is required for basic auth method")
	}

	b := &basic{
		sqlClient: opts.SQLClient,
		oasDir:    opts.OASDir,
		config:    opts.AuthZConfig,
	}

	return b, nil
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
	session, err := dbGetSessionRow(req.Context(), a.sqlClient, sessionID)
	if errors.Is(err, errNoSession) {
		zerologr.Error(apierror.ErrNoSession, "Failed to find a matching session")
		return apierror.APIErrNoSession
	}
	if err != nil {
		return apierror.APIErrInternal
	}

	if time.Now().UnixMilli() > session.Expires {
		zerologr.Error(apierror.ErrNoSession, "Session expired")
		return apierror.APIErrNoSession
	}

	req.Header.Set("X-Krb-Org", strconv.Itoa(int(session.OrgID)))
	req.Header.Set("X-Krb-User", strconv.Itoa(int(session.UserID)))

	return nil
}

//nolint:gocognit // welp
func (a *basic) Authorized(req *http.Request) error {
	zerologr.V(50).Info("Authorizing request " + req.URL.Path)
	//nolint:errcheck // bigger problems if this is missing
	backend := req.Context().Value(composer.BackendContextKey).(string)

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
	orgID, err := strconv.ParseInt(req.Header.Get("X-Krb-Org"), 10, 64)
	if err != nil {
		zerologr.Error(err, "Failed to parse org ID header")
		return apierror.APIErrInternal
	}
	userID, err := strconv.ParseInt(req.Header.Get("X-Krb-User"), 10, 64)
	if err != nil {
		zerologr.Error(err, "Failed to parse user ID header")
		return apierror.APIErrInternal
	}

	userGroups, err := dbGetUserGroupNames(req.Context(), a.sqlClient, orgID, userID)
	if err != nil {
		return apierror.APIErrInternal
	}

	for _, g := range userGroups {
		req.Header.Add("X-Krb-Groups", g)
	}

	for _, usergroup := range userGroups {
		if slices.Contains(groupsToValidate, usergroup) {
			return nil
		}
	}

	// No group match found -> 403
	return apierror.APIErrForbidden
}

// RegisterRoutes registers the API routes for the basic auth method.
func (a *basic) RegisterRoutes(
	mux *http.ServeMux,
	middleware ...authbasicapi.StrictMiddlewareFunc,
) error {
	data, err := os.ReadFile(fmt.Sprintf("%s/%s", a.oasDir, authBasicSpecification))
	if err != nil {
		return fmt.Errorf("failed to read basic authentication OAS: %w", err)
	}

	spec, err := openapi3.NewLoader().LoadFromData(data)
	if err != nil {
		return fmt.Errorf("failed to load basic authentication OAS: %w", err)
	}

	ssi := newSSI(a.sqlClient)
	authMiddleware := make([]authbasicapi.StrictMiddlewareFunc, len(middleware)+1)
	authMiddleware[0] = AuthMiddleware(ssi)

	for i := range middleware {
		authMiddleware[i+1] = middleware[i]
	}

	strictHandler := authbasicapi.NewStrictHandlerWithOptions(
		ssi,
		authMiddleware,
		authbasicapi.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  apierror.RequestErrorHandler,
			ResponseErrorHandlerFunc: apierror.ResponseErrorHandler,
		},
	)

	_ = authbasicapi.HandlerWithOptions(strictHandler, authbasicapi.StdHTTPServerOptions{
		BaseRouter: mux,
		Middlewares: []authbasicapi.MiddlewareFunc{
			oas.ValidationMiddleware(spec),
			func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					zerologr.Info(fmt.Sprintf("%s %s", r.Method, r.URL.Path))
					next.ServeHTTP(w, r)
				})
			},
		},
	})

	return nil
}

func applySchemas(sqlClient db.SQLClient) error {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), db.SchemaApplyTimeout)
	defer cancel()
	if _, err := sqlClient.Exec(timeoutCtx, string(dbschemaBytes)); err != nil {
		return err
	}
	return nil
}
