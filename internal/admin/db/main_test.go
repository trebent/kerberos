//go:build !postgres_integration

package admindb

import (
	"os"
	"testing"

	"github.com/trebent/kerberos/internal/db"
	"github.com/trebent/kerberos/internal/db/sqlite"
)

const (
	testClientID     = "dummy-client-id"
	testClientSecret = "dummy-client-secret"
)

var testClient db.SQLClient

func TestMain(m *testing.M) {
	testClient = sqlite.New(&sqlite.Opts{DSN: "test.db"})
	if err := ApplySchemas(testClient); err != nil {
		panic("failed to apply admin DB schema: " + err.Error())
	}

	if err := BootstrapSuperuser(testClient, testClientID, testClientSecret); err != nil {
		panic("failed to bootstrap superuser: " + err.Error())
	}

	if err := BootstrapPermissions(testClient); err != nil {
		panic("failed to bootstrap permissions: " + err.Error())
	}

	code := m.Run()

	_ = os.Remove("test.db")

	os.Exit(code)
}
