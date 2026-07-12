//go:build !postgres_integration

package basic

import (
	"context"
	"testing"

	authbasicapi "github.com/trebent/kerberos/internal/oapi/auth/basic"
)

// TestBasicSSIRefreshNoRefreshCookie verifies that Refresh returns 401 when the context
// contains no refresh token (simulates a missing refresh cookie).
func TestBasicSSIRefreshNoRefreshCookie(t *testing.T) {
	ssi := newSSI(testClient)

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
	ssi := newSSI(testClient)

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
