//go:build postgres_integration

package admin

import (
	"fmt"
	"os"
	"testing"

	"github.com/trebent/kerberos/internal/db"
	"github.com/trebent/kerberos/internal/db/postgres"
)

const (
	testClientID     = "dummy-client-id"
	testClientSecret = "dummy-client-secret"
)

var testClient db.SQLClient

func postgresDSN() string {
	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		return dsn
	}
	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "localhost"
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
		panic("failed to apply admin DB schema: " + err.Error())
	}

	if err := dbBootstrapSuperuser(testClient, testClientID, testClientSecret); err != nil {
		panic("failed to bootstrap superuser: " + err.Error())
	}

	if err := dbBootstrapPermissions(testClient); err != nil {
		panic("failed to bootstrap permissions: " + err.Error())
	}

	os.Exit(m.Run())
}
