package basic

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"github.com/trebent/kerberos/internal/composer"
	"github.com/trebent/kerberos/internal/config"
	authbasicapi "github.com/trebent/kerberos/internal/oapi/auth/basic"
)

func TestAuthorizer_Authenticated(t *testing.T) {
	basic, err := New(&Opts{
		AuthZConfig: map[string]*config.AuthZ{},
		SQLClient:   testClient,
		OASDir:      "something",
	})
	if err != nil {
		t.Fatal("Expected no error when creating authorizer")
	}

	req, err := http.NewRequest("GET", "/api/v1/some/path", nil)
	if err != nil {
		t.Fatal("Expected no error when creating request")
	}

	// Test that the authorizer returns an error when no cookie is present.
	if err := basic.Authenticated(req); err == nil {
		t.Fatal("Expected an error when no cookie is present")
	}

	// Create a known session in the database.
	orgId, adminId := mustCreateOrg(t, uniqueName(t, "authN-test-org"))
	if err := dbCreateSession(req.Context(), testClient, adminId, orgId, "session"); err != nil {
		t.Fatal("Expected no error when creating session")
	}

	// Add the cookie to the request.
	req.AddCookie(&http.Cookie{Name: "session", Value: "session"})

	// Test that the authorizer returns no error when a valid cookie is present.
	if err := basic.Authenticated(req); err != nil {
		t.Fatal("Expected no error when a valid cookie is present")
	}

	// Add an invalid cookie to the request.
	req, err = http.NewRequest("GET", "/api/v1/some/path", nil)
	if err != nil {
		t.Fatal("Expected no error when creating request")
	}
	req.AddCookie(&http.Cookie{Name: "session", Value: "invalid"})

	// Test that the authorizer returns an error when an invalid cookie is present.
	if err := basic.Authenticated(req); err == nil {
		t.Fatal("Expected an error when an invalid cookie is present")
	}
}

func TestAuthorizer_AuthorizedGroup(t *testing.T) {
	groupName := uniqueName(t, "authZ-admin")
	basic, err := New(&Opts{
		AuthZConfig: map[string]*config.AuthZ{
			"backend": {
				Groups: []string{groupName},
			},
		},
		SQLClient: testClient,
		OASDir:    "something",
	})
	if err != nil {
		t.Fatal("Expected no error when creating authorizer")
	}

	orgID, _ := mustCreateOrg(t, uniqueName(t, "authZ-test-org"))
	userID := mustCreateUser(t, orgID, uniqueName(t, "authZ-test-user"))
	groupID := mustCreateGroup(t, orgID, groupName)
	if err := dbUpdateUserGroupBindings(
		t.Context(),
		testClient,
		orgID,
		userID,
		[]authbasicapi.Group{{Id: groupID, Name: groupName}}); err != nil {
		t.Fatalf("dbUpdateUserGroupBindings error: %v", err)
	}

	if err := dbCreateSession(t.Context(), testClient, userID, orgID, "session"); err != nil {
		t.Fatal("Expected no error when creating session")
	}

	req, err := http.NewRequest("GET", "/api/v1/some/path", nil)
	if err != nil {
		t.Fatal("Expected no error when creating request")
	}
	req.AddCookie(&http.Cookie{Name: "session", Value: "session"})
	req.Header.Add("X-Krb-Org", strconv.Itoa(int(orgID)))
	req.Header.Add("X-Krb-User", strconv.Itoa(int(userID)))
	req = req.WithContext(context.WithValue(req.Context(), composer.BackendContextKey, "backend"))

	if err := basic.Authorized(req); err != nil {
		t.Fatal("Expected no error when user is authorized")
	}

	if err := dbUpdateUserGroupBindings(
		t.Context(),
		testClient,
		orgID,
		userID,
		[]authbasicapi.Group{},
	); err != nil {
		t.Fatalf("dbUpdateUserGroupBindings error: %v", err)
	}

	if err := basic.Authorized(req); err == nil {
		t.Fatal("Expected an error when user is not authorized")
	}
}
