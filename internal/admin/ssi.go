package admin

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	adminext "github.com/trebent/kerberos/internal/admin/extensions"
	adminapi "github.com/trebent/kerberos/internal/api/admin"
	"github.com/trebent/kerberos/internal/db"
	"github.com/trebent/zerologr"
	"golang.org/x/time/rate"
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
		DB db.SQLClient

		ClientID     string
		ClientSecret string
	}
	impl struct {
		db db.SQLClient

		sessionCleaner *time.Ticker

		superSessions sync.Map
		limiter       *rate.Limiter

		clientID     string
		clientSecret string

		flowFetcher adminext.FlowFetcher
		oasBackend  adminext.OASBackend
	}
)

const (
	superSessionCleanerInterval = 1 * time.Minute
	superSessionExpiry          = 15 * time.Minute
	limiterRate                 = 1 * time.Second
	// TODO: make configurable
	limiterMaxBurst = 100
)

var (
	_ withExtensions = (*impl)(nil)

	errSuperUserRateLimited = errors.New("rate limiter does not permit action")
)

func makeGenAPIError(msg string) adminapi.APIErrorResponse {
	return adminapi.APIErrorResponse{Errors: []string{msg}}
}

func newSSI(opts *ssiOpts) withExtensions {
	i := &impl{
		db: opts.DB,

		sessionCleaner: time.NewTicker(superSessionCleanerInterval),

		superSessions: sync.Map{},
		limiter:       rate.NewLimiter(rate.Every(limiterRate), limiterMaxBurst),

		clientID:     opts.ClientID,
		clientSecret: opts.ClientSecret,
	}

	go func(im *impl) {
		for range im.sessionCleaner.C {
			im.superSessions.Range(func(key, value any) bool {
				t, _ := value.(time.Time)
				if time.Now().After(t) {
					zerologr.V(20).Info("Cleaning up an expired super user session")
					im.superSessions.Delete(key)
				}

				return true
			})
		}
	}(i)
	return i
}

func (i *impl) SetFlowFetcher(ff adminext.FlowFetcher) {
	i.flowFetcher = ff
}

func (i *impl) SetOASBackend(ob adminext.OASBackend) {
	i.oasBackend = ob
}

// LoginSuperuser implements [StrictServerInterface].
func (i *impl) LoginSuperuser(
	_ context.Context,
	request adminapi.LoginSuperuserRequestObject,
) (adminapi.LoginSuperuserResponseObject, error) {
	//nolint:nilerr // on purpose
	if !i.limiter.Allow() {
		zerologr.Error(errSuperUserRateLimited, "Super user login is being rate-limited")
		return adminapi.LoginSuperuser429JSONResponse(
			makeGenAPIError(http.StatusText(http.StatusTooManyRequests)),
		), nil
	}

	if i.clientID != request.Body.ClientId || i.clientSecret != request.Body.ClientSecret {
		zerologr.Info("User login failed due to password mismatch")
		return adminapi.LoginSuperuser401JSONResponse(makeGenAPIError("Login failed.")), nil
	}

	sessionID := uuid.NewString()
	i.superSessions.Store(sessionID, time.Now().Add(superSessionExpiry))

	return adminapi.LoginSuperuser204Response{
		Headers: adminapi.LoginSuperuser204ResponseHeaders{
			XKrbSession: sessionID,
		},
	}, nil
}

// LogoutSuperuser implements [StrictServerInterface].
func (i *impl) LogoutSuperuser(
	ctx context.Context,
	_ adminapi.LogoutSuperuserRequestObject,
) (adminapi.LogoutSuperuserResponseObject, error) {
	i.superSessions.Delete(SessionIDFromContext(ctx))
	return adminapi.LogoutSuperuser204Response{}, nil
}

// GetFlow implements [adminapi.StrictServerInterface].
func (i *impl) GetFlow(
	_ context.Context,
	_ adminapi.GetFlowRequestObject,
) (adminapi.GetFlowResponseObject, error) {
	// No need to nil-check flowFetcher since KRB won't be able to start without it.
	return adminapi.GetFlow200JSONResponse(i.flowFetcher.GetFlow()), nil
}

// GetBackendOAS implements [adminapi.StrictServerInterface].
func (i *impl) GetBackendOAS(
	_ context.Context,
	request adminapi.GetBackendOASRequestObject,
) (adminapi.GetBackendOASResponseObject, error) {
	// Must nil-check OAS backend since it's an optional extension and not required for KRB to start.
	if i.oasBackend == nil {
		return adminapi.GetBackendOAS404JSONResponse(makeGenAPIError("not configured")), nil
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
