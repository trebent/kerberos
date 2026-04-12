package admin

import (
	"os"
	"testing"

	"github.com/trebent/kerberos/internal/db/sqlite"
)

func TestMain(m *testing.M) {
	sqlClient := sqlite.New(&sqlite.Opts{DSN: "test.db"})
	if err := applySchemas(sqlClient); err != nil {
		panic("failed to apply admin DB schema: " + err.Error())
	}

	code := m.Run()

	_ = os.Remove("test.db")

	os.Exit(code)
}
