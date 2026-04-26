//go:build postgres_integration

package postgres_test

import (
	"database/sql"
	"os"
	"testing"

	"github.com/trebent/kerberos/internal/db/postgres"
)

func dsn(t *testing.T) string {
	t.Helper()
	d := os.Getenv("POSTGRES_DSN")
	if d == "" {
		d = "host=localhost port=5432 dbname=kerberos user=kerberos password=kerberos sslmode=disable"
	}
	return d
}

func TestPostgres(t *testing.T) {
	db := postgres.New(&postgres.Opts{DSN: dsn(t)})

	_, err := db.Exec(t.Context(), "CREATE TABLE IF NOT EXISTS _test_pg (id SERIAL PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}
	defer func() {
		_, _ = db.Exec(t.Context(), "DROP TABLE IF EXISTS _test_pg")
	}()

	_, err = db.Exec(t.Context(), "INSERT INTO _test_pg (name) VALUES ($1)", "Alice")
	if err != nil {
		t.Fatalf("failed to insert: %v", err)
	}

	_, err = db.Exec(t.Context(), "INSERT INTO _test_pg (name) VALUES ($1)", "Bob")
	if err != nil {
		t.Fatalf("failed to insert: %v", err)
	}

	rows, err := db.Query(t.Context(), "SELECT id, name FROM _test_pg ORDER BY id")
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			t.Fatalf("failed to scan: %v", err)
		}
		count++
		t.Logf("Row: id=%d name=%s", id, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 rows, got %d", count)
	}
}

func TestPostgres_NamedArgs(t *testing.T) {
	db := postgres.New(&postgres.Opts{DSN: dsn(t)})

	_, err := db.Exec(t.Context(), "CREATE TABLE IF NOT EXISTS _test_pg_named (id SERIAL PRIMARY KEY, name TEXT NOT NULL)")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	defer func() {
		_, _ = db.Exec(t.Context(), "DROP TABLE IF EXISTS _test_pg_named")
	}()

	_, err = db.Exec(t.Context(),
		"INSERT INTO _test_pg_named (name) VALUES (@name)",
		sql.NamedArg{Name: "name", Value: "Charlie"},
	)
	if err != nil {
		t.Fatalf("insert with named arg: %v", err)
	}

	rows, err := db.Query(t.Context(),
		"SELECT id, name FROM _test_pg_named WHERE name = @name",
		sql.NamedArg{Name: "name", Value: "Charlie"},
	)
	if err != nil {
		t.Fatalf("query with named arg: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("expected one row")
	}
	var id int
	var name string
	if err := rows.Scan(&id, &name); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if name != "Charlie" {
		t.Fatalf("expected Charlie, got %q", name)
	}
}
