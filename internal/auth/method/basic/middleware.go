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
	"github.com/trebent/kerberos/internal/security"
	"github.com/trebent/zerologr"
)

type contextKey string

var (
	userContextKey    contextKey = "user"
	sessionContextKey contextKey = "session"
	refreshContextKey contextKey = "refresh"

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
				if r.Method != http.MethodGet {
					zerologr.Info(
						"basicauthorgviewer denied non-GET method access",
						"method",
						r.Method,
					)
					return nil, apierror.ErrForbidden
				}
				return f(ctx, w, r, request)
			}

			if len(r.Cookies()) == 0 {
				zerologr.V(20).Info("No cookies found, denying access")
				return nil, apierror.ErrUnauthenticated
			}

			if len(r.CookiesNamed(security.RefreshCookieName)) == 0 {
				zerologr.V(20).Info("No refresh cookie found")
			} else {
				ctx = context.WithValue(
					ctx,
					refreshContextKey,
					r.CookiesNamed(security.RefreshCookieName)[0].Value,
				)
			}

			if operationID == "Refresh" {
				zerologr.V(20).Info("Permitting refresh path access")
				return f(ctx, w, r, request)
			}

			if len(r.CookiesNamed(security.SessionCookieName)) == 0 {
				zerologr.V(20).Info("No session cookie found, denying access")
				return nil, apierror.ErrUnauthenticated
			}

			cookie := r.CookiesNamed(security.SessionCookieName)[0]
			if cookie.Value == "" {
				zerologr.V(20).Info("Session cookie is empty, denying access")
				return nil, apierror.ErrUnauthenticated
			}

			session, err := dbGetSessionRow(ctx, apiImpl.db, cookie.Value)
			if errors.Is(err, errNoSession) {
				zerologr.Error(apierror.ErrUnauthenticated, "Failed to find a matching session")
				return nil, apierror.ErrUnauthenticated
			}
			if err != nil {
				return nil, apierror.ErrISE
			}

			if time.Now().UnixMilli() > session.Expires {
				zerologr.Error(apierror.ErrUnauthenticated, "Session expired")
				return nil, apierror.ErrUnauthenticated
			}

			var validation []error
			switch operationID {
			case "CreateOrganisation", "ListOrganisations":
				zerologr.V(20).Info("Validating creating/listing organisations")
				validation = make([]error, 1)
				validation[0] = apierror.ErrForbidden
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
			ctx = withSession(ctx, session.SessionID)
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

func withSession(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionContextKey, sessionID)
}

func sessionFromContext(ctx context.Context) string {
	//nolint:errcheck // welp
	return ctx.Value(sessionContextKey).(string)
}

func orgValidator(orgID int64, r *http.Request) error {
	parsedOrgID, err := strconv.ParseInt(r.PathValue("orgID"), 10, 0)
	if err != nil {
		return apierror.New(http.StatusBadRequest, errMalformedOrgID.Error())
	}

	if parsedOrgID != orgID {
		return apierror.ErrForbidden
	}

	return nil
}

func administratorValidator(isAdministrator bool) error {
	if isAdministrator {
		return nil
	}

	return apierror.ErrForbidden
}

func ownerUserValidator(userID int64, r *http.Request) error {
	parsedUserID, err := strconv.ParseInt(r.PathValue("userID"), 10, 0)
	if err != nil {
		return apierror.New(http.StatusBadRequest, errMalformedUserID.Error())
	}

	if parsedUserID != userID {
		return apierror.ErrForbidden
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
