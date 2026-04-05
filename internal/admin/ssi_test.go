package admin

import (
	"errors"
	"testing"

	apierror "github.com/trebent/kerberos/internal/oapi/error"
)

func TestAdminDummyOASBackend(t *testing.T) {
	admin := newSSI(&ssiOpts{
		SQLClient:    nil,
		ClientID:     "dummy-client-id",
		ClientSecret: "dummy-client-secret",
	}).(*impl)

	_, err := admin.oasBackend.GetOAS("dummy-backend")
	if !errors.Is(err, apierror.APIErrNotFound) {
		t.Fatalf("expected APIErrNotFound, got %v", err)
	}
}
