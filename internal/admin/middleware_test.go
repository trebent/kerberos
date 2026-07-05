package admin

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/trebent/kerberos/internal/admin/model"
	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
	apierror "github.com/trebent/kerberos/internal/oapi/error"
	"github.com/trebent/kerberos/internal/security"
)

func TestAdminSessionMiddleware(t *testing.T) {
	ssi, err := newSSI(&ssiOpts{
		SQLClient:    testClient,
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
	})
	if err != nil {
		t.Fatalf("expected newSSI to succeed, got error: %v", err)
	}
	ssiImpl := ssi.(*impl)

	mw := SessionMiddleware(ssiImpl)

	wg := sync.WaitGroup{}
	wg.Add(1)
	handler := mw(func(ctx context.Context, w http.ResponseWriter, r *http.Request, req any) (any, error) {
		if !ContextSessionValid(ctx) {
			t.Fatalf("Expected valid session")
		}

		if !IsSuperUserContext(ctx) {
			t.Fatalf("Expected super user context")
		}

		wg.Done()
		return nil, nil
	}, "")

	// Force login to create a session.
	response, err := ssiImpl.LoginSuperuser(t.Context(), adminapi.LoginSuperuserRequestObject{
		Body: &adminapi.LoginSuperuserJSONRequestBody{
			ClientId:     testClientID,
			ClientSecret: testClientSecret,
		},
	})
	if err != nil {
		t.Fatalf("Failed to login superuser: %v", err)
	}

	decodedResponse, ok := response.(customSuperLoginResponse)
	if !ok {
		t.Fatalf("Expected customSuperLoginResponse, got %T", response)
	}

	var sessionCookie *http.Cookie
	for _, c := range decodedResponse.cookies {
		cookie, err := http.ParseSetCookie(c)
		if err != nil {
			t.Fatalf("Failed to parse cookie: %v", err)
		}

		if cookie.Name == security.SessionCookieName {
			sessionCookie = cookie
			break
		}
	}

	req, err := http.NewRequest(http.MethodGet, "/", bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.AddCookie(sessionCookie)
	_, err = handler(
		t.Context(),
		httptest.NewRecorder(),
		req,
		adminapi.LoginSuperuserRequestObject{},
	)
	wg.Wait()

	if err != nil {
		t.Fatalf("Did not expect error: %v", err)
	}
}

func TestAdminRequireSessionMiddleware(t *testing.T) {
	mw := RequireSessionMiddleware()

	handler := mw(func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
		return nil, nil
	}, "operation")

	enrichedContext := context.Background()
	enrichedContext = context.WithValue(
		enrichedContext, adminContextSession, &model.Session{UserID: 1, SessionID: "123", Expires: time.Now().Add(time.Hour).UnixMilli()},
	)
	_, err := handler(enrichedContext, httptest.NewRecorder(), &http.Request{}, adminapi.LoginSuperuserRequestObject{})
	if err != nil {
		t.Fatalf("Did not expect error: %v", err)
	}

	emptyContext := context.Background()
	_, err = handler(emptyContext, httptest.NewRecorder(), &http.Request{}, adminapi.LoginSuperuserJSONRequestBody{})
	if err == nil {
		t.Fatal("Expected an error")
	}

	if !errors.Is(err, apierror.ErrUnauthenticated) {
		t.Fatal("Expected an apierror")
	}
}
