package adminapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	adminapi "github.com/trebent/kerberos/internal/api/admin"
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
