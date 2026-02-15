package adminapi

import (
	"context"
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

		superSessions sync.Map
		limiter       *rate.Limiter
		clientID      string
		clientSecret  string
	}
)

const (
	superSessionExpiry = 15 * time.Minute
	limiterRate        = 1 * time.Second
	limiterMaxBurst    = 10
)

func makeGenAPIError(msg string) adminapi.APIErrorResponse {
	return adminapi.APIErrorResponse{Errors: []string{msg}}
}

func NewSSI(db db.SQLClient, clientID, clientSecret string) adminapi.StrictServerInterface {
	return &impl{
		db:            db,
		superSessions: sync.Map{},
		limiter:       rate.NewLimiter(rate.Every(limiterRate), limiterMaxBurst),
		clientID:      clientID,
		clientSecret:  clientSecret,
	}
}

// LoginSuperuser implements [StrictServerInterface].
func (i *impl) LoginSuperuser(
	ctx context.Context,
	request adminapi.LoginSuperuserRequestObject,
) (adminapi.LoginSuperuserResponseObject, error) {
	//nolint:nilerr // on purpose
	if err := i.limiter.Wait(ctx); err != nil {
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
	_ context.Context,
	_ adminapi.LogoutSuperuserRequestObject,
) (adminapi.LogoutSuperuserResponseObject, error) {
	i.superSessions.Delete(i.clientID)
	return adminapi.LogoutSuperuser204Response{}, nil
}
