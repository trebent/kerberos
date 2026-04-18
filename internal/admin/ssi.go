package admin

import (
	"bytes"
	"context"
	"errors"

	adminext "github.com/trebent/kerberos/internal/admin/extensions"
	"github.com/trebent/kerberos/internal/db"
	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
	apierror "github.com/trebent/kerberos/internal/oapi/error"
	"github.com/trebent/zerologr"
)

type (
	withExtensions interface {
		adminapi.StrictServerInterface

		// Extensions.

		// SetFlowFetcher sets the flow fetcher for the SSI, allowing it to serve flow metadata to the admin API.
		SetFlowFetcher(adminext.FlowFetcher)
		// SetOASBackend sets the OAS backend for the SSI, allowing it to serve OAS data to the admin API.
		SetOASBackend(adminext.OASBackend)
	}
	ssiOpts struct {
		SQLClient db.SQLClient

		ClientID     string
		ClientSecret string
	}
	impl struct {
		sqlClient db.SQLClient

		flowFetcher adminext.FlowFetcher
		oasBackend  adminext.OASBackend
	}
)

var (
	_ withExtensions = (*impl)(nil)

	errNoSuperuser = errors.New("no superuser exists")
	errNoSession   = errors.New("no valid super session found")

	apiErrInternal = adminapi.InternalErrorJSONResponse(
		makeGenAPIError(apierror.APIErrInternal.Error()),
	)
	apiErrForbidden = adminapi.ForbiddenErrorJSONResponse(
		makeGenAPIError(apierror.ErrNoPermission.Error()),
	)
	apiErrNotFound = adminapi.NotFoundErrorJSONResponse(
		makeGenAPIError(apierror.ErrNotFound.Error()),
	)
	apiErrConflict = adminapi.ConflictErrorJSONResponse(
		makeGenAPIError(apierror.ErrConflict.Error()),
	)
)

func makeGenAPIError(msg string) adminapi.APIErrorResponse {
	return adminapi.APIErrorResponse{Errors: []string{msg}}
}

func makeErrUnauthorized(msg string) adminapi.UnauthorizedErrorJSONResponse {
	return adminapi.UnauthorizedErrorJSONResponse(makeGenAPIError(msg))
}

func newSSI(opts *ssiOpts) (withExtensions, error) {
	i := &impl{
		sqlClient: opts.SQLClient,

		oasBackend: &adminext.DummyOASBackend{},
	}

	if err := dbBootstrapSuperuser(i.sqlClient, opts.ClientID, opts.ClientSecret); err != nil {
		return nil, err
	}

	if err := dbBootstrapPermissions(i.sqlClient); err != nil {
		return nil, err
	}

	return i, nil
}

func (i *impl) SetFlowFetcher(ff adminext.FlowFetcher) {
	i.flowFetcher = ff
}

func (i *impl) SetOASBackend(ob adminext.OASBackend) {
	i.oasBackend = ob
}

// GetFlow implements [adminapi.StrictServerInterface].
func (i *impl) GetFlow(
	ctx context.Context,
	_ adminapi.GetFlowRequestObject,
) (adminapi.GetFlowResponseObject, error) {
	if !IsSuperUserContext(ctx) && !ContextCanViewFlow(ctx) {
		return adminapi.GetFlow403JSONResponse{
			ForbiddenErrorJSONResponse: apiErrForbidden,
		}, nil
	}

	return adminapi.GetFlow200JSONResponse(i.flowFetcher.GetFlow()), nil
}

// GetBackendOAS implements [adminapi.StrictServerInterface].
func (i *impl) GetBackendOAS(
	ctx context.Context,
	request adminapi.GetBackendOASRequestObject,
) (adminapi.GetBackendOASResponseObject, error) {
	if i.oasBackend == nil {
		return adminapi.GetBackendOAS404JSONResponse(apiErrNotFound), nil
	}

	if !IsSuperUserContext(ctx) && !ContextCanViewOAS(ctx) {
		return adminapi.GetBackendOAS403JSONResponse(apiErrForbidden), nil
	}

	oasData, err := i.oasBackend.GetOAS(request.Backend)
	if err != nil {
		return nil, err
	}

	return adminapi.GetBackendOAS200ApplicationyamlResponse{
		Body:          bytes.NewReader(oasData),
		ContentLength: int64(len(oasData)),
	}, nil
}

// GetPermissions implements [adminapi.StrictServerInterface].
func (i *impl) GetPermissions(
	ctx context.Context,
	_ adminapi.GetPermissionsRequestObject,
) (adminapi.GetPermissionsResponseObject, error) {
	perms, err := dbListPermissions(ctx, i.sqlClient)
	if err != nil {
		zerologr.Error(err, "Failed to list admin permissions")
		return adminapi.GetPermissions500JSONResponse{
			InternalErrorJSONResponse: apiErrInternal,
		}, nil
	}

	return adminapi.GetPermissions200JSONResponse(perms), nil
}
