package admin

import (
	"errors"
	"testing"

	"github.com/trebent/kerberos/internal/db/sqlite"
	adminapi "github.com/trebent/kerberos/internal/oapi/admin"
	apierror "github.com/trebent/kerberos/internal/oapi/error"
)

func TestAdminSSIDummyOASBackend(t *testing.T) {
	sqlClient := sqlite.New(&sqlite.Opts{DSN: "test.db"})
	applySchemas(sqlClient)
	ssi := newSSI(&ssiOpts{
		SQLClient:    sqlClient,
		ClientID:     "dummy-client-id",
		ClientSecret: "dummy-client-secret",
	}).(*impl)

	_, err := ssi.oasBackend.GetOAS("dummy-backend")
	if !errors.Is(err, apierror.APIErrNotFound) {
		t.Fatalf("expected APIErrNotFound, got %v", err)
	}
}

func TestAdminSSISuperuserBootstrap(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("bootstrapSuperuser panicked: %v", r)
		}
	}()

	sqlClient := sqlite.New(&sqlite.Opts{DSN: "test.db"})
	applySchemas(sqlClient)
	ssi := newSSI(&ssiOpts{
		SQLClient:    sqlClient,
		ClientID:     "dummy-client-id",
		ClientSecret: "dummy-client-secret",
	}).(*impl)

	// Check if superuser was created.
	superuser, err := ssi.querySuperuser()
	if err != nil {
		t.Fatalf("expected superuser to be created, got error: %v", err)
	}

	if superuser.Username != "dummy-client-id" {
		t.Fatalf("expected superuser username to be %s, got %s", "dummy-client-id", superuser.Username)
	}
}

func TestAdminSSISuperuser(t *testing.T) {
	sqlClient := sqlite.New(&sqlite.Opts{DSN: "test.db"})
	applySchemas(sqlClient)
	ssi := newSSI(&ssiOpts{
		SQLClient:    sqlClient,
		ClientID:     "dummy-client-id",
		ClientSecret: "dummy-client-secret",
	})

	_, err := ssi.LoginSuperuser(t.Context(), adminapi.LoginSuperuserRequestObject{
		Body: &adminapi.LoginSuperuserJSONRequestBody{
			ClientId:     "dummy-client-id",
			ClientSecret: "dummy-client-secret",
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
