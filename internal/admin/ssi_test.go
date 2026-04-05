package admin

import (
	"errors"
	"testing"

	"github.com/trebent/kerberos/internal/db/sqlite"
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

func TestAdminSSSISuperuserBootstrap(t *testing.T) {
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
