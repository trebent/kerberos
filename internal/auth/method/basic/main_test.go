//go:build !postgres_integration

package basic

import (
	"os"
	"testing"

	"github.com/trebent/kerberos/internal/db"
	"github.com/trebent/kerberos/internal/db/sqlite"
)

var testClient db.SQLClient

func TestMain(m *testing.M) {
	testClient = sqlite.New(&sqlite.Opts{DSN: "test.db"})
	if err := applySchemas(testClient); err != nil {
		panic("failed to apply basic auth DB schema: " + err.Error())
	}

	code := m.Run()

	_ = os.Remove("test.db")

	os.Exit(code)
}
