//go:build !postgres_integration

package basic

import (
	"context"
	"testing"

	"github.com/trebent/kerberos/internal/config"
	authbasicapi "github.com/trebent/kerberos/internal/oapi/auth/basic"
)

// TestBasicSSIRefreshNoRefreshCookie verifies that Refresh returns 401 when the context
// contains no refresh token (simulates a missing refresh cookie).
func TestBasicSSIRefreshNoRefreshCookie(t *testing.T) {
	ssi := newSSI(testClient, &config.Cookies{})

	resp, err := ssi.Refresh(t.Context(), authbasicapi.RefreshRequestObject{OrgID: 0})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if _, ok := resp.(authbasicapi.Refresh401JSONResponse); !ok {
		t.Fatalf("expected Refresh401JSONResponse, got %T", resp)
	}
}

// TestBasicSSIRefresh verifies that Refresh succeeds when the context contains a refresh token
// linked to a valid session. No session context is needed — only the refresh token.
func TestBasicSSIRefresh(t *testing.T) {
	ssi := newSSI(testClient, &config.Cookies{})

	orgID, userID := mustCreateOrg(t, uniqueName(t, "ssi-refresh-org"))

	refreshID := uniqueName(t, "refresh-basic")
	sessionID := uniqueName(t, "session-basic-refresh")
	if err := dbCreateSession(t.Context(), testClient, userID, orgID, refreshID, sessionID); err != nil {
		t.Fatalf("dbCreateSession error: %v", err)
	}

	// Inject only the refresh token — no session context — to prove session context is not required.
	ctx := context.WithValue(t.Context(), refreshContextKey, refreshID)

	resp, err := ssi.Refresh(ctx, authbasicapi.RefreshRequestObject{OrgID: orgID})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if _, ok := resp.(customRefreshSessionResponse); !ok {
		t.Fatalf("expected customRefreshSessionResponse, got %T", resp)
	}
}

// TestBasicSSINoUserSession verifies that GetMe and Logout return 401 instead of panicking when the
// middleware let the request through without a user session, as it does for admin sessions.
func TestBasicSSINoUserSession(t *testing.T) {
	ssi := newSSI(testClient, &config.Cookies{})

	me, err := ssi.GetMe(t.Context(), authbasicapi.GetMeRequestObject{OrgID: 1})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if _, ok := me.(authbasicapi.GetMe401JSONResponse); !ok {
		t.Fatalf("expected GetMe401JSONResponse, got %T", me)
	}

	logout, err := ssi.Logout(t.Context(), authbasicapi.LogoutRequestObject{OrgID: 1})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if _, ok := logout.(authbasicapi.Logout401JSONResponse); !ok {
		t.Fatalf("expected Logout401JSONResponse, got %T", logout)
	}
}
