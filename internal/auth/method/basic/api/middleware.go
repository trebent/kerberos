//nolint:gocognit // ?
package basicapi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	authbasicapi "github.com/trebent/kerberos/internal/api/auth/basic"
	apierror "github.com/trebent/kerberos/internal/api/error"
	"github.com/trebent/zerologr"
)

type contextKey string

var (
	userContextKey contextKey = "user"

	errMalformedOrgID  = errors.New("malformed organisation ID")
	errWrongOrgID      = errors.New("organisation ID mismatch")
	errMalformedUserID = errors.New("malformed user ID")
	errWrongUserID     = errors.New("user ID mismatch")
)

//nolint:funlen // welp
func AuthMiddleware(ssi authbasicapi.StrictServerInterface) authbasicapi.StrictMiddlewareFunc {
	//nolint:errcheck // welp
	apiImpl := ssi.(*impl)

	return func(f authbasicapi.StrictHandlerFunc, operationID string) authbasicapi.StrictHandlerFunc {
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

			sessionID := r.Header.Get("X-Krb-Session")
			if sessionID == "" {
				zerologr.V(20).Info("Failed to find a session header")

				if zerologr.V(30).Enabled() {
					for key, values := range r.Header {
						zerologr.V(30).Info("Header "+key, "values", values)
					}
				}
				return nil, apierror.APIErrNoSession
			}

			rows, err := apiImpl.db.Query(
				ctx,
				queryGetSession,
				sql.NamedArg{Name: "sessionID", Value: sessionID},
			)
			if err != nil {
				zerologr.Error(err, "Failed to query session")
				return nil, apierror.APIErrInternal
			}

			if !rows.Next() {
				if err := rows.Err(); err != nil {
					zerologr.Error(err, "Failed to load next row")
					return nil, apierror.APIErrInternal
				}

				zerologr.Error(apierror.APIErrNoSession, "Failed to find a matching session")
				return nil, apierror.APIErrNoSession
			}

			var (
				sessionUserID int64
				sessionOrgID  int64
				administrator bool
				superUser     bool
				expires       int64
			)
			err = rows.Scan(&sessionUserID, &sessionOrgID, &administrator, &superUser, &expires)
			//nolint:sqlclosecheck // won't help here
			_ = rows.Close()
			if err != nil {
				zerologr.Error(err, "Failed to scan row")
				return nil, apierror.APIErrInternal
			}

			if time.Now().UnixMilli() > expires {
				zerologr.Error(apierror.ErrNoSession, "Session expired")
				return nil, apierror.APIErrNoSession
			}

			if superUser {
				zerologr.Info(
					fmt.Sprintf("Permitting super user access to operation %s", operationID),
				)
				return f(ctx, w, r, request)
			}

			var validation []error
			switch operationID {
			case "CreateOrganisation", "ListOrganisations":
				zerologr.Info("Validating creating/listing organisations")
				validation = make([]error, 1)
				validation[0] = apierror.APIErrForbidden
			case "Logout":
				zerologr.V(20).Info("Validating auth for logout path")
				validation = make([]error, 1)
				validation[0] = orgValidator(sessionOrgID, r)
			case
				"CreateUser",
				"ListUsers":
				zerologr.V(20).Info("Validating auth for user paths")
				validation = make([]error, 2)
				validation[0] = orgValidator(sessionOrgID, r)
				validation[1] = administratorValidator(administrator)
			case "GetUser":
				zerologr.V(20).Info("Validating auth for GET user path")
				validation = make([]error, 2)
				validation[0] = orgValidator(sessionOrgID, r)
				validation[1] = or(
					administratorValidator(administrator),
					ownerUserValidator(sessionUserID, r),
				)
			case
				"UpdateUser",
				"DeleteUser",
				"GetUserGroups",
				"UpdateUserGroups",
				"ChangePassword":
				zerologr.V(20).Info("Validating auth for specific user paths")
				validation = make([]error, 2)
				validation[0] = orgValidator(sessionOrgID, r)
				validation[1] = administratorValidator(administrator)
			case
				"GetOrganisation",
				"DeleteOrganisation":
				zerologr.V(20).Info("Validating auth for specific org paths")
				validation = make([]error, 2)
				validation[0] = orgValidator(sessionOrgID, r)
				validation[1] = administratorValidator(administrator)
			case
				"CreateGroup",
				"ListGroups":
				zerologr.V(20).Info("Validating auth for group paths")
				validation = make([]error, 2)
				validation[0] = orgValidator(sessionOrgID, r)
				validation[1] = administratorValidator(administrator)
			case
				"UpdateGroup",
				"GetGroup",
				"DeleteGroup":
				zerologr.V(20).Info("Validating auth for specific group paths")
				validation = make([]error, 2)
				validation[0] = orgValidator(sessionOrgID, r)
				validation[1] = administratorValidator(administrator)
			default:
				validation = make([]error, 1)
				validation[0] = apierror.New(
					http.StatusNotFound,
					fmt.Sprintf("%v: %s", errors.ErrUnsupported, operationID),
				)
			}

			if err := errors.Join(validation...); err != nil {
				zerologr.V(10).Info("Validation failed", "err", err, "url", r.URL.Path)
				return nil, err
			}

			ctx = withUser(ctx, sessionUserID)
			return f(ctx, w, r, request)
		}
	}
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
		return apierror.New(http.StatusBadRequest, errMalformedOrgID.Error())
	}

	if parsedOrgID != orgID {
		return apierror.New(http.StatusForbidden, errWrongOrgID.Error())
	}

	return nil
}

func administratorValidator(isAdministrator bool) error {
	if isAdministrator {
		return nil
	}

	return apierror.APIErrForbidden
}

func ownerUserValidator(userID int64, r *http.Request) error {
	parsedUserID, err := strconv.ParseInt(r.PathValue("userID"), 10, 0)
	if err != nil {
		return apierror.New(http.StatusBadRequest, errMalformedUserID.Error())
	}

	if parsedUserID != userID {
		return apierror.New(http.StatusForbidden, errWrongUserID.Error())
	}

	return nil
}

// or is a permissive error iterator. If any error in the input list is nil,
// or will return nil.
func or(errs ...error) error {
	for _, err := range errs {
		if err == nil {
			return nil
		}
	}

	return errors.Join(errs...)
}
