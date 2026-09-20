package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/trebent/kerberos/internal/auth/method/basic"
)

func TestContextExtractors(t *testing.T) {
	req := &http.Request{
		Header: http.Header{
			basic.HeaderUser:   []string{"user"},
			basic.HeaderOrg:    []string{"12"},
			basic.HeaderGroups: []string{"g1", "g2"},
		},
	}

	called := false
	handler := ContextMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		user, present := UserFromContext(r.Context())
		if !present {
			t.Fatal("User was not present")
		}
		if user != "user" {
			t.Fatalf("Unexpected User value: %s", user)
		}

		org, present := OrgFromContext(r.Context())
		if !present {
			t.Fatal("Org was not present")
		}
		if org != "12" {
			t.Fatalf("Unexpected Org value: %s", user)
		}

		groups, present := GroupsFromContext(r.Context())
		if !present {
			t.Fatal("Groups was not present")
		}
		if len(groups) != 2 {
			t.Fatalf("Unexpected group count: %d", len(groups))
		}
	}))

	handler.ServeHTTP(httptest.NewRecorder(), req)

	if !called {
		t.Fatal("Handler was not called")
	}
}
