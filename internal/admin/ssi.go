package admin

import (
	"bytes"
	"context"
	"errors"
	"time"

	adminext "github.com/trebent/kerberos/internal/admin/extensions"
	"github.com/trebent/kerberos/internal/db"
	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
	apierror "github.com/trebent/kerberos/internal/oapi/error"
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

const (
	superSessionExpiry = 15 * time.Minute
)

var (
	_ withExtensions = (*impl)(nil)

	errNoSuperuser = errors.New("no superuser exists")
	errNoSession   = errors.New("no valid super session found")
)

func makeGenAPIError(msg string) adminapi.APIErrorResponse {
	return adminapi.APIErrorResponse{Errors: []string{msg}}
}

func makeErrInternal() adminapi.InternalErrorJSONResponse {
	return adminapi.InternalErrorJSONResponse(makeGenAPIError(apierror.APIErrInternal.Error()))
}

func makeErrNotFound() adminapi.NotFoundErrorJSONResponse {
	return adminapi.NotFoundErrorJSONResponse(makeGenAPIError(apierror.ErrNotFound.Error()))
}

func makeErrUnauthorized(msg string) adminapi.UnauthorizedErrorJSONResponse {
	return adminapi.UnauthorizedErrorJSONResponse(makeGenAPIError(msg))
}

func newSSI(opts *ssiOpts) withExtensions {
	i := &impl{
		sqlClient: opts.SQLClient,

		oasBackend: &adminext.DummyOASBackend{},
	}
	i.bootstrapSuperuser(opts.ClientID, opts.ClientSecret)
	return i
}

func (i *impl) SetFlowFetcher(ff adminext.FlowFetcher) {
	i.flowFetcher = ff
}

func (i *impl) SetOASBackend(ob adminext.OASBackend) {
	i.oasBackend = ob
}

// GetFlow implements [adminapi.StrictServerInterface].
func (i *impl) GetFlow(
	_ context.Context,
	_ adminapi.GetFlowRequestObject,
) (adminapi.GetFlowResponseObject, error) {
	return adminapi.GetFlow200JSONResponse(i.flowFetcher.GetFlow()), nil
}

// GetBackendOAS implements [adminapi.StrictServerInterface].
func (i *impl) GetBackendOAS(
	_ context.Context,
	request adminapi.GetBackendOASRequestObject,
) (adminapi.GetBackendOASResponseObject, error) {
	oasData, err := i.oasBackend.GetOAS(request.Backend)
	if err != nil {
		return nil, err
	}

	return adminapi.GetBackendOAS200ApplicationyamlResponse{
		Body:          bytes.NewReader(oasData),
		ContentLength: int64(len(oasData)),
	}, nil
}
