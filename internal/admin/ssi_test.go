package admin

import (
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
	if !errors.Is(err, apierror.APIErrNotFound) {
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
