package api

import (
	"context"
	"net/http"

	"github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
	"github.com/trebent/zerologr"
)

var AuthMiddleware StrictMiddlewareFunc = func(
	f nethttp.StrictHTTPHandlerFunc,
	operationID string,
) nethttp.StrictHTTPHandlerFunc {
	return func(
		ctx context.Context,
		w http.ResponseWriter,
		r *http.Request,
		request any) (response any, err error) {

		switch operationID {
		case "Login", "Logout":
			zerologr.V(20).Info("Skipping authentication for login/logout paths")
		case "CreateUser", "ListUsers", "GetUser", "UpdateUser", "DeleteUser", "UpdateUserGroups", "ChangePassword":
			zerologr.V(20).Info("Validating auth for user paths")
		case "CreateOrganisation", "ListOrganisations", "UpdateOrganisation", "GetOrganisation", "DeleteOrganisation":
			zerologr.V(20).Info("Validating auth for org paths")
		case "CreateGroup", "ListGroups", "UpdateGroup", "GetGroup", "DeleteGroup":
			zerologr.V(20).Info("Validating auth for group paths")
		}

		return f(ctx, w, r, request)
	}
}
