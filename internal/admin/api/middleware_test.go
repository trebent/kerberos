package adminapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	adminapi "github.com/trebent/kerberos/internal/api/admin"
	apierror "github.com/trebent/kerberos/internal/api/error"
	"github.com/trebent/kerberos/internal/db/sqlite"
)

func TestAdminSessionMiddleware(t *testing.T) {
	ssi := NewSSI(sqlite.New(&sqlite.Opts{DSN: "test.db"}), "admin", "secret")
	impl := ssi.(*impl)
	impl.superSessions.Store("session", time.Now().Add(1*time.Hour))
	mw := AdminSessionMiddleware(ssi)

	wg := sync.WaitGroup{}
	wg.Add(1)
	handler := mw(func(ctx context.Context, w http.ResponseWriter, r *http.Request, req any) (any, error) {
		sessionID := SessionIDFromContext(ctx)
		if sessionID != "session" {
			t.Fatalf("Expected session ID %s, got %s", "session", sessionID)
		}

		if !IsSuperUserContext(ctx) {
			t.Fatalf("Expected super user context")
		}

		wg.Done()
		return nil, nil
	}, "")

	headers := http.Header{}
	headers.Add("x-krb-session", "session")
	_, err := handler(
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

func TestRequireSessionMiddleware(t *testing.T) {
	mw := RequireSessionMiddleware()

	handler := mw(func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
		return nil, nil
	}, "operation")

	enrichedContext := context.Background()
	enrichedContext = context.WithValue(enrichedContext, adminContextSession, "session")
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
