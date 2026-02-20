package adminapi

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	adminapi "github.com/trebent/kerberos/internal/api/admin"
	"github.com/trebent/kerberos/internal/db"
	"github.com/trebent/zerologr"
	"golang.org/x/time/rate"
)

type (
	impl struct {
		db db.SQLClient

		sessionCleaner *time.Ticker
		superSessions  sync.Map
		limiter        *rate.Limiter
		clientID       string
		clientSecret   string
	}
)

const (
	superSessionCleanerInterval = 1 * time.Minute
	superSessionExpiry          = 15 * time.Minute
	limiterRate                 = 1 * time.Second
	limiterMaxBurst             = 10
)

var errSuperUserRateLimited = errors.New("rate limiter does not permit action")

func makeGenAPIError(msg string) adminapi.APIErrorResponse {
	return adminapi.APIErrorResponse{Errors: []string{msg}}
}

func NewSSI(db db.SQLClient, clientID, clientSecret string) adminapi.StrictServerInterface {
	i := &impl{
		db:             db,
		sessionCleaner: time.NewTicker(superSessionCleanerInterval),
		superSessions:  sync.Map{},
		limiter:        rate.NewLimiter(rate.Every(limiterRate), limiterMaxBurst),
		clientID:       clientID,
		clientSecret:   clientSecret,
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
