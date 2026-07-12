package admin

import (
	"context"
	"errors"
	"testing"

	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
	apierror "github.com/trebent/kerberos/internal/oapi/error"
)

func TestAdminSSIDummyOASBackend(t *testing.T) {
	ssi, err := newSSI(&ssiOpts{
		SQLClient:    testClient,
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
	})
	if err != nil {
		t.Fatalf("expected newSSI to succeed, got error: %v", err)
	}
	ssiImpl := ssi.(*impl)

	_, err = ssiImpl.oasBackend.GetOAS("dummy-backend")
	if !errors.Is(err, apierror.ErrNotFound) {
		t.Fatalf("expected APIErrNotFound, got %v", err)
	}
}

func TestAdminSSISuperuserBootstrap(t *testing.T) {
	_, err := newSSI(&ssiOpts{
		SQLClient:    testClient,
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
	})
	if err != nil {
		t.Fatalf("expected newSSI to succeed, got error: %v", err)
	}

	// Check if superuser was created.
	superuser, err := dbGetSuperuser(t.Context(), testClient)
	if err != nil {
		t.Fatalf("expected superuser to be created, got error: %v", err)
	}

	if superuser.Username != testClientID {
		t.Fatalf("expected superuser username to be %s, got %s", testClientID, superuser.Username)
	}
}

func TestAdminSSIPermissionBootstrap(t *testing.T) {
	_, err := newSSI(&ssiOpts{
		SQLClient:    testClient,
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
	})
	if err != nil {
		t.Fatalf("expected newSSI to succeed, got error: %v", err)
	}

	// Check if permissions were created.
	permissions, err := dbListPermissions(t.Context(), testClient)
	if err != nil {
		t.Fatalf("expected permissions to be created, got error: %v", err)
	}

	if len(permissions) == 0 {
		t.Fatalf("expected permissions to be created, got none")
	}
}

func TestAdminSSISuperuser(t *testing.T) {
	ssi, err := newSSI(&ssiOpts{
		SQLClient:    testClient,
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
	})
	if err != nil {
		t.Fatalf("expected newSSI to succeed, got error: %v", err)
	}

	// Test superuser login.

	_, err = ssi.LoginSuperuser(t.Context(), adminapi.LoginSuperuserRequestObject{
		Body: &adminapi.LoginSuperuserJSONRequestBody{
			ClientId:     testClientID,
			ClientSecret: testClientSecret,
		},
	})
	if err != nil {
		t.Fatalf("expected superuser login to succeed, got error: %v", err)
	}

	_, err = ssi.LogoutSuperuser(t.Context(), adminapi.LogoutSuperuserRequestObject{})
	if err != nil {
		t.Fatalf("expected superuser logout to succeed, got error: %v", err)
	}
}

// TestAdminSSIRefreshSuperuserSessionNoRefreshCookie verifies that RefreshSuperuserSession
// returns 401 when the context contains no refresh token (simulates a missing refresh cookie).
func TestAdminSSIRefreshSuperuserSessionNoRefreshCookie(t *testing.T) {
	ssi, err := newSSI(&ssiOpts{
		SQLClient:    testClient,
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
	})
	if err != nil {
		t.Fatalf("expected newSSI to succeed, got error: %v", err)
	}

	resp, err := ssi.RefreshSuperuserSession(t.Context(), adminapi.RefreshSuperuserSessionRequestObject{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if _, ok := resp.(adminapi.RefreshSuperuserSession401JSONResponse); !ok {
		t.Fatalf("expected RefreshSuperuserSession401JSONResponse, got %T", resp)
	}
}

// TestAdminSSIRefreshSuperuserSession verifies that RefreshSuperuserSession succeeds when the
// context contains a refresh token linked to a superuser session. No session context is needed —
// only the refresh token.
func TestAdminSSIRefreshSuperuserSession(t *testing.T) {
	ssi, err := newSSI(&ssiOpts{
		SQLClient:    testClient,
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
	})
	if err != nil {
		t.Fatalf("expected newSSI to succeed, got error: %v", err)
	}

	superuser, err := dbGetSuperuser(t.Context(), testClient)
	if err != nil {
		t.Fatalf("expected superuser to exist, got error: %v", err)
	}

	refreshID := uniqueName(t, "refresh-super")
	sessionID := uniqueName(t, "session-super")
	if err := dbCreateSession(t.Context(), testClient, superuser.ID, refreshID, sessionID); err != nil {
		t.Fatalf("dbCreateSession error: %v", err)
	}

	// Inject only the refresh token — no session context — to prove session context is not required.
	ctx := context.WithValue(t.Context(), adminContextRefresh, refreshID)

	resp, err := ssi.RefreshSuperuserSession(ctx, adminapi.RefreshSuperuserSessionRequestObject{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if _, ok := resp.(customRefreshSuperuserSessionResponse); !ok {
		t.Fatalf("expected customRefreshSuperuserSessionResponse, got %T", resp)
	}
}

// TestAdminSSIRefreshSuperuserSessionForbidden verifies that RefreshSuperuserSession returns 403
// when the refresh token belongs to a non-superuser session.
func TestAdminSSIRefreshSuperuserSessionForbidden(t *testing.T) {
	ssi, err := newSSI(&ssiOpts{
		SQLClient:    testClient,
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
	})
	if err != nil {
		t.Fatalf("expected newSSI to succeed, got error: %v", err)
	}

	userID := mustCreateAdminUser(t, uniqueName(t, "user-refresh-forbidden"))
	refreshID := uniqueName(t, "refresh-forbidden")
	sessionID := uniqueName(t, "session-forbidden")
	if err := dbCreateSession(t.Context(), testClient, userID, refreshID, sessionID); err != nil {
		t.Fatalf("dbCreateSession error: %v", err)
	}

	ctx := context.WithValue(t.Context(), adminContextRefresh, refreshID)

	resp, err := ssi.RefreshSuperuserSession(ctx, adminapi.RefreshSuperuserSessionRequestObject{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if _, ok := resp.(adminapi.RefreshSuperuserSession403JSONResponse); !ok {
		t.Fatalf("expected RefreshSuperuserSession403JSONResponse, got %T", resp)
	}
}

// TestAdminSSIRefreshUserSessionNoRefreshCookie verifies that RefreshUserSession returns 401
// when the context contains no refresh token (simulates a missing refresh cookie).
func TestAdminSSIRefreshUserSessionNoRefreshCookie(t *testing.T) {
	ssi, err := newSSI(&ssiOpts{
		SQLClient:    testClient,
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
	})
	if err != nil {
		t.Fatalf("expected newSSI to succeed, got error: %v", err)
	}

	resp, err := ssi.RefreshUserSession(t.Context(), adminapi.RefreshUserSessionRequestObject{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if _, ok := resp.(adminapi.RefreshUserSession401JSONResponse); !ok {
		t.Fatalf("expected RefreshUserSession401JSONResponse, got %T", resp)
	}
}

// TestAdminSSIRefreshUserSession verifies that RefreshUserSession succeeds when the context
// contains a refresh token linked to a non-superuser admin session. No session context is needed —
// only the refresh token.
func TestAdminSSIRefreshUserSession(t *testing.T) {
	ssi, err := newSSI(&ssiOpts{
		SQLClient:    testClient,
		ClientID:     testClientID,
		ClientSecret: testClientSecret,
	})
	if err != nil {
		t.Fatalf("expected newSSI to succeed, got error: %v", err)
	}

	userID := mustCreateAdminUser(t, uniqueName(t, "user-refresh-ok"))
	refreshID := uniqueName(t, "refresh-user")
	sessionID := uniqueName(t, "session-user-refresh")
	if err := dbCreateSession(t.Context(), testClient, userID, refreshID, sessionID); err != nil {
		t.Fatalf("dbCreateSession error: %v", err)
	}

	// Inject only the refresh token — no session context.
	ctx := context.WithValue(t.Context(), adminContextRefresh, refreshID)

	resp, err := ssi.RefreshUserSession(ctx, adminapi.RefreshUserSessionRequestObject{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if _, ok := resp.(customRefreshSessionResponse); !ok {
		t.Fatalf("expected customRefreshSessionResponse, got %T", resp)
	}
}
