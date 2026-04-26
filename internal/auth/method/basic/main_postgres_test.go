//go:build postgres_integration

package basic

import (
	"fmt"
	"os"
	"testing"

	"github.com/trebent/kerberos/internal/db"
	"github.com/trebent/kerberos/internal/db/postgres"
)

var testClient db.SQLClient

func postgresDSN() string {
	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		return dsn
	}
	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "localhost:5432"
	}
	dbName := os.Getenv("POSTGRES_DB")
	if dbName == "" {
		dbName = "kerberos"
	}
	user := os.Getenv("POSTGRES_USER")
	if user == "" {
		user = "kerberos"
	}
	password := os.Getenv("POSTGRES_PASSWORD")
	if password == "" {
		password = "kerberos"
	}
	return fmt.Sprintf("host=%s dbname=%s user=%s password=%s sslmode=disable", host, dbName, user, password)
}

func TestMain(m *testing.M) {
	testClient = postgres.New(&postgres.Opts{DSN: postgresDSN()})
	if err := applySchemas(testClient); err != nil {
		panic("failed to apply basic auth DB schema: " + err.Error())
	}

	os.Exit(m.Run())
}
