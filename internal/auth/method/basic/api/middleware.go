//nolint:gocognit // ?
package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
	"github.com/trebent/zerologr"
)

type contextKey string

var (
	orgContextKey  contextKey = "org"
	userContextKey contextKey = "user"

	ErrNoSession = errors.New("no session found")
	ErrInternal  = errors.New("internal error")
)

func AuthMiddleware(ssi StrictServerInterface) StrictMiddlewareFunc {
	//nolint:errcheck // welp
	apiImpl := ssi.(*impl)

	return func(f nethttp.StrictHTTPHandlerFunc, operationID string) nethttp.StrictHTTPHandlerFunc {
		return func(
			ctx context.Context,
			w http.ResponseWriter,
			r *http.Request,
			request any,
		) (any, error) {
			zerologr.V(20).Info("Running basic auth API middleware", "url", r.URL.Path)

			// No middleware operations needed for logging in or out.
			if operationID == "Login" || operationID == "Logout" {
				zerologr.V(20).Info("Skipping authentication for login/logout paths")
				return f(ctx, w, r, request)
			}

			if operationID == "CreateOrganisation" {
				zerologr.Info("Validating superuser credentials for creating organisations")
				// TODO: implement superuser credentials validation.
				return f(ctx, w, r, request)
			}

			sessionID := r.Header.Get("X-Krb-Session")
			if sessionID == "" {
				zerologr.Error(ErrNoSession, "Failed to find a session header")
				for key, values := range r.Header {
					zerologr.Info("Header "+key, "values", values)
				}

				w.WriteHeader(http.StatusUnauthorized)
				return nil, ErrNoSession
			}

			rows, err := apiImpl.db.Query(
				ctx,
				queryGetSession,
				sql.NamedArg{Name: "sessionID", Value: sessionID},
			)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return nil, ErrInternal
			}

			if !rows.Next() {
				if err := rows.Err(); err != nil {
					zerologr.Error(err, "Failed to load next row")
					w.WriteHeader(http.StatusInternalServerError)
					return nil, ErrInternal
				}

				zerologr.Error(ErrNoSession, "Failed to find a matching session")
				w.WriteHeader(http.StatusUnauthorized)
				return nil, ErrNoSession
			}

			var (
				userID         int64
				organisationID int64
			)
			err = rows.Scan(&userID, &organisationID, new(string), new(int64))
			//nolint:sqlclosecheck // won't help here
			_ = rows.Close()
			if err != nil {
				zerologr.Error(err, "Failed to scan row")
				return GenericErrorResponse{Message: "Internal error."}, nil
			}

			ctx = withOrg(ctx, organisationID)
			ctx = withUser(ctx, userID)
			switch operationID {
			case "CreateUser",
				"ListUsers",
				"GetUser",
				"UpdateUser",
				"DeleteUser",
				"UpdateUserGroups",
				"ChangePassword":
				zerologr.V(20).Info("Validating auth for user paths")
			case "ListOrganisations", "UpdateOrganisation", "GetOrganisation", "DeleteOrganisation":
				zerologr.V(20).Info("Validating auth for org paths")
			case "CreateGroup", "ListGroups", "UpdateGroup", "GetGroup", "DeleteGroup":
				zerologr.V(20).Info("Validating auth for group paths")
			}

			return f(ctx, w, r, request)
		}
	}
}

func withOrg(ctx context.Context, orgID int64) context.Context {
	return context.WithValue(ctx, orgContextKey, orgID)
}

func orgFromContext(ctx context.Context) int64 {
	//nolint:errcheck // welp
	return ctx.Value(orgContextKey).(int64)
}

func withUser(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userContextKey, userID)
}

func userFromContext(ctx context.Context) int64 {
	//nolint:errcheck // welp
	return ctx.Value(userContextKey).(int64)
}
