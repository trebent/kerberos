package admin

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
	if err := applySchemas(testClient); err != nil {
		panic("failed to apply admin DB schema: " + err.Error())
	}

	dbBootstrapSuperuser(testClient, testClientID, testClientSecret)
	dbBootstrapPermissions(testClient)

	code := m.Run()

	_ = os.Remove("test.db")

	os.Exit(code)
}
