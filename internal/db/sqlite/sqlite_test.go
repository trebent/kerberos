package sqlite_test

import (
	"os"
	"testing"

	"github.com/trebent/kerberos/internal/db/sqlite"
)

func TestSQLite(t *testing.T) {
	dsn := "test.db"
	defer func() {
		if err := os.Remove(dsn); err != nil {
			t.Fatalf("failed to remove test database: %v", err)
		}
	}()

	db := sqlite.New(&sqlite.Opts{DSN: dsn})

	_, err := db.Exec(t.Context(), "CREATE TABLE IF NOT EXISTS test (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}

	_, err = db.Exec(t.Context(), "INSERT INTO test (name) VALUES (?)", "Alice")
	if err != nil {
		t.Fatalf("failed to insert into test table: %v", err)
	}

	_, err = db.Exec(t.Context(), "INSERT INTO test (name) VALUES (?)", "Bob")
	if err != nil {
		t.Fatalf("failed to insert into test table: %v", err)
	}

	rows, err := db.Query(t.Context(), "SELECT id, name FROM test")
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			t.Fatalf("failed to scan row: %v", err)
		}
		count++
		t.Logf("Row: id=%d, name=%s", id, name)
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("rows error: %v", err)
	}

	if count != 2 {
		t.Fatalf("expected 2 rows, got %d", count)
	}
}

func TestSQLite_ForeignKey(t *testing.T) {
	dsn := "test_fk.db"
	defer func() {
		if err := os.Remove(dsn); err != nil {
			t.Fatalf("failed to remove test database: %v", err)
		}
	}()

	db := sqlite.New(&sqlite.Opts{DSN: dsn})

	_, err := db.Exec(t.Context(), "CREATE TABLE IF NOT EXISTS parent (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("failed to create parent table: %v", err)
	}

	_, err = db.Exec(t.Context(), "CREATE TABLE IF NOT EXISTS child (id INTEGER PRIMARY KEY, parent_id INTEGER, name TEXT, FOREIGN KEY(parent_id) REFERENCES parent(id) ON DELETE CASCADE)")
	if err != nil {
		t.Fatalf("failed to create child table: %v", err)
	}

	_, err = db.Exec(t.Context(), "INSERT INTO parent (name) VALUES (?)", "Parent1")
	if err != nil {
		t.Fatalf("failed to insert into parent table: %v", err)
	}

	_, err = db.Exec(t.Context(), "INSERT INTO child (parent_id, name) VALUES (?, ?)", 1, "Child1")
	if err != nil {
		t.Fatalf("failed to insert into child table: %v", err)
	}

	rows, err := db.Query(t.Context(), "SELECT c.id, c.name, p.name FROM child c JOIN parent p ON c.parent_id = p.id")
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int
		var childName, parentName string
		if err := rows.Scan(&id, &childName, &parentName); err != nil {
			t.Fatalf("failed to scan row: %v", err)
		}
		count++
		t.Logf("Row: id=%d, child_name=%s, parent_name=%s", id, childName, parentName)

		if childName != "Child1" || parentName != "Parent1" {
			t.Fatalf("unexpected data: child_name=%s, parent_name=%s", childName, parentName)
		}
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("rows error: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected 1 row, got %d", count)
	}

	_, err = db.Exec(t.Context(), "DELETE FROM parent WHERE id = ?", 1)
	if err != nil {
		t.Fatalf("failed to delete from parent table: %v", err)
	}

	rows, err = db.Query(t.Context(), "SELECT id FROM child")
	if err != nil {
		t.Fatalf("failed to query child table: %v", err)
	}
	defer rows.Close()

	if rows.Next() {
		t.Fatalf("expected no rows in child table after parent deletion")
	}
}
