package admin

import (
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

	decodedResponse, ok := response.(adminapi.LoginSuperuser204Response)
	if !ok {
		t.Fatalf("Expected LoginSuperuser204Response, got %T", response)
	}

	headers := http.Header{}
	headers.Add("x-krb-session", decodedResponse.Headers.XKrbSession)
	_, err = handler(
		t.Context(),
		httptest.NewRecorder(),
		&http.Request{
			Header: headers,
		},
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

	if !errors.Is(err, apierror.APIErrNoSession) {
		t.Fatal("Expected an apierror")
	}
}

func TestAdminContextHasPermission(t *testing.T) {
	// Superuser context should have all permissions.
	superUserContext := context.WithValue(context.Background(), adminContextIsSuperUser, true)
	if !ContextHasPermission(superUserContext, 1) {
		t.Fatal("Expected superuser context to have all permissions")
	}

	// Context with specific permissions should report correctly.
	permIDs := []int64{1, 2, 3}
	permContext := context.WithValue(context.Background(), adminContextPermissions, permIDs)
	if !ContextHasPermission(permContext, 2) {
		t.Fatal("Expected context to have permission ID 2")
	}
	if ContextHasPermission(permContext, 4) {
		t.Fatal("Expected context to not have permission ID 4")
	}

	// Context with no permissions should report false.
	noPermContext := context.WithValue(context.Background(), adminContextPermissions, []int64{})
	if ContextHasPermission(noPermContext, 1) {
		t.Fatal("Expected context with no permissions to not have permission ID 1")
	}

	// Context with invalid permissions type should report false.
	invalidPermContext := context.WithValue(context.Background(), adminContextPermissions, "invalid")
	if ContextHasPermission(invalidPermContext, 1) {
		t.Fatal("Expected context with invalid permissions type to not have permission ID 1")
	}

	// Context with nil permissions should report false.
	nilPermContext := context.WithValue(context.Background(), adminContextPermissions, nil)
	if ContextHasPermission(nilPermContext, 1) {
		t.Fatal("Expected context with nil permissions to not have permission ID 1")
	}
}
