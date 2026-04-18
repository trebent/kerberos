//nolint:gocognit // ?
package basic

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/trebent/kerberos/internal/admin"
	authbasicapi "github.com/trebent/kerberos/internal/oapi/auth/basic"
	apierror "github.com/trebent/kerberos/internal/oapi/error"
	"github.com/trebent/zerologr"
)

type contextKey string

var (
	userContextKey contextKey = "user"

	errMalformedOrgID  = errors.New("malformed organisation ID")
	errMalformedUserID = errors.New("malformed user ID")
)

//nolint:funlen // welp
func AuthMiddleware(ssi authbasicapi.StrictServerInterface) authbasicapi.StrictMiddlewareFunc {
	apiImpl, ok := ssi.(*impl)
	if !ok {
		panic("expected auth api *impl")
	}

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

			if admin.IsSuperUserContext(ctx) {
				zerologr.V(20).Info("Permitting super user access")
				return f(ctx, w, r, request)
			}

			if admin.ContextIsBasicAuthAdmin(ctx) {
				zerologr.V(20).Info("Permitting basicauthorgadmin access")
				return f(ctx, w, r, request)
			}

			if admin.ContextIsBasicAuthViewer(ctx) {
				zerologr.V(20).Info("Validating basicauthorgviewer access")
				if r.Method != "GET" {
					zerologr.V(20).Info("basicauthorgviewer denied non-GET method", "method", r.Method)
					return nil, apierror.APIErrForbidden
				}
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

			session, err := dbGetSessionRow(ctx, apiImpl.db, sessionID)
			if errors.Is(err, errNoSession) {
				zerologr.Error(apierror.APIErrNoSession, "Failed to find a matching session")
				return nil, apierror.APIErrNoSession
			}
			if err != nil {
				return nil, apierror.APIErrInternal
			}

			if time.Now().UnixMilli() > session.Expires {
				zerologr.Error(apierror.ErrNoSession, "Session expired")
				return nil, apierror.APIErrNoSession
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
				validation[0] = orgValidator(session.OrgID, r)
			case
				"CreateUser",
				"ListUsers":
				zerologr.V(20).Info("Validating auth for user paths")
				validation = make([]error, 2)
				validation[0] = orgValidator(session.OrgID, r)
				validation[1] = administratorValidator(session.Administrator)
			case "GetUser":
				zerologr.V(20).Info("Validating auth for GET user path")
				validation = make([]error, 2)
				validation[0] = orgValidator(session.OrgID, r)
				validation[1] = or(
					administratorValidator(session.Administrator),
					ownerUserValidator(session.UserID, r),
				)
			case
				"UpdateUser",
				"DeleteUser",
				"GetUserGroups",
				"ChangePassword":
				zerologr.V(20).Info("Validating auth for user owned paths")
				validation = make([]error, 2)
				validation[0] = orgValidator(session.OrgID, r)
				validation[1] = or(
					administratorValidator(session.Administrator),
					ownerUserValidator(session.UserID, r),
				)
			case "UpdateUserGroups":
				zerologr.V(20).Info("Validating auth for update user membership paths")
				validation = make([]error, 2)
				validation[0] = orgValidator(session.OrgID, r)
				validation[1] = administratorValidator(session.Administrator)
			case
				"GetOrganisation",
				"DeleteOrganisation":
				zerologr.V(20).Info("Validating auth for specific org paths")
				validation = make([]error, 2)
				validation[0] = orgValidator(session.OrgID, r)
				validation[1] = administratorValidator(session.Administrator)
			case
				"CreateGroup",
				"ListGroups":
				zerologr.V(20).Info("Validating auth for group paths")
				validation = make([]error, 2)
				validation[0] = orgValidator(session.OrgID, r)
				validation[1] = administratorValidator(session.Administrator)
			case
				"UpdateGroup",
				"GetGroup",
				"DeleteGroup":
				zerologr.V(20).Info("Validating auth for specific group paths")
				validation = make([]error, 2)
				validation[0] = orgValidator(session.OrgID, r)
				validation[1] = administratorValidator(session.Administrator)
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

			ctx = withUser(ctx, session.UserID)
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
		return apierror.APIErrForbidden
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
		return apierror.APIErrForbidden
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
