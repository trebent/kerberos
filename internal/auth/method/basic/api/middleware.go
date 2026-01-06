//nolint:gocognit // ?
package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
	"github.com/trebent/zerologr"
)

type contextKey string

var (
	orgContextKey  contextKey = "org"
	userContextKey contextKey = "user"

	ErrNoSession      = errors.New("no session found")
	ErrInternal       = errors.New("internal error")
	ErrMalformedOrgID = errors.New("malformed organisation ID")
	ErrWrongOrgID     = errors.New("organisation ID mismatch")

	//nolint:errname // This is intentional to separate pure error types from wrapper API Errors.
	APIErrNoSession = &Error{message: ErrNoSession.Error(), StatusCode: http.StatusUnauthorized}
	//nolint:errname // This is intentional to separate pure error types from wrapper API Errors.
	APIErrInternal = &Error{
		message:    ErrInternal.Error(),
		StatusCode: http.StatusInternalServerError,
	}
)

//nolint:funlen // welp
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

			// No middleware operations needed for logging in.
			if operationID == "Login" {
				zerologr.V(20).Info("Skipping authentication for the login path")
				return f(ctx, w, r, request)
			}

			if operationID == "CreateOrganisation" || operationID == "ListOrganisations" {
				zerologr.Info("Validating superuser credentials for creating/listing organisations")
				// TODO: implement superuser credentials validation.
				return f(ctx, w, r, request)
			}

			sessionID := r.Header.Get("X-Krb-Session")
			if sessionID == "" {
				zerologr.Error(ErrNoSession, "Failed to find a session header")
				for key, values := range r.Header {
					zerologr.Info("Header "+key, "values", values)
				}

				return nil, APIErrNoSession
			}

			rows, err := apiImpl.db.Query(
				ctx,
				queryGetSession,
				sql.NamedArg{Name: "sessionID", Value: sessionID},
			)
			if err != nil {
				zerologr.Error(err, "Failed to query session")
				return nil, APIErrInternal
			}

			if !rows.Next() {
				if err := rows.Err(); err != nil {
					zerologr.Error(err, "Failed to load next row")
					return nil, APIErrInternal
				}

				zerologr.Error(ErrNoSession, "Failed to find a matching session")
				return nil, APIErrNoSession
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
				return nil, APIErrInternal
			}

			ctx = withOrg(ctx, sessionOrgID)
			ctx = withUser(ctx, sessionUserID)
			validation := make([]error, 1)
			switch operationID {
			case "Logout":
				zerologr.V(20).Info("Validating auth for logout path")
				validation[0] = orgValidator(sessionOrgID, r)
			case
				"CreateUser",
				"ListUsers":
				zerologr.V(20).Info("Validating auth for user paths")
				validation[0] = orgValidator(sessionOrgID, r)
			case
				"GetUser",
				"UpdateUser",
				"DeleteUser",
				"GetUserGroups",
				"UpdateUserGroups",
				"ChangePassword":
				zerologr.V(20).Info("Validating auth for specific user paths")
				validation[0] = orgValidator(sessionOrgID, r)
			case
				"GetOrganisation",
				"DeleteOrganisation":
				zerologr.V(20).Info("Validating auth for specific org paths")
				validation[0] = orgValidator(sessionOrgID, r)
			case
				"CreateGroup",
				"ListGroups":
				zerologr.V(20).Info("Validating auth for group paths")
				validation[0] = orgValidator(sessionOrgID, r)
			case
				"UpdateGroup",
				"GetGroup",
				"DeleteGroup":
				zerologr.V(20).Info("Validating auth for specific group paths")
				validation[0] = orgValidator(sessionOrgID, r)
			default:
				validation[0] = fmt.Errorf("%w: %s", errors.ErrUnsupported, operationID)
			}

			if err := errors.Join(validation...); err != nil {
				zerologr.V(10).Info("Validation failed", "err", err, "url", r.URL.Path)
				return nil, err
			}

			return f(ctx, w, r, request)
		}
	}
}

func withOrg(ctx context.Context, orgID int64) context.Context {
	return context.WithValue(ctx, orgContextKey, orgID)
}

//nolint:unused // evaluate if this is needed
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

func orgValidator(orgID int64, r *http.Request) error {
	parsedOrgID, err := strconv.ParseInt(r.PathValue("orgID"), 10, 0)
	if err != nil {
		return &Error{
			StatusCode: http.StatusBadRequest,
			message:    ErrMalformedOrgID.Error(),
		}
	}

	if parsedOrgID != orgID {
		return &Error{
			StatusCode: http.StatusForbidden,
			message:    ErrWrongOrgID.Error(),
		}
	}

	return nil
}
