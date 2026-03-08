package adminapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	adminapi "github.com/trebent/kerberos/internal/api/admin"
	"github.com/trebent/kerberos/internal/composer"
	"github.com/trebent/kerberos/internal/db"
	"github.com/trebent/zerologr"
	"golang.org/x/time/rate"
)

type (
	SSI interface {
		adminapi.StrictServerInterface

		SetComposer(composer.Composer)
	}
	Opts struct {
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

		composer composer.Composer
	}
	flowMetaResponse struct {
		FlowMeta []*composer.FlowMeta
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
	_ SSI                            = (*impl)(nil)
	_ adminapi.GetFlowResponseObject = (*flowMetaResponse)(nil)

	errSuperUserRateLimited = errors.New("rate limiter does not permit action")
)

func (fmr *flowMetaResponse) VisitGetFlowResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(fmr.FlowMeta); err != nil {
		return err
	}

	w.WriteHeader(http.StatusOK)
	return nil
}

func makeGenAPIError(msg string) adminapi.APIErrorResponse {
	return adminapi.APIErrorResponse{Errors: []string{msg}}
}

func NewSSI(opts *Opts) SSI {
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

func (i *impl) SetComposer(c composer.Composer) {
	i.composer = c
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
	return &flowMetaResponse{FlowMeta: i.composer.GetFlowMeta()}, nil
}
