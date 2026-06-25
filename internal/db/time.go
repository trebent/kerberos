package db

import (
	"fmt"
	"time"
)

// TimeString implements [database/sql.Scanner] for time.Time values stored as
// RFC3339/RFC3339Nano strings (e.g. by the SQLite driver).
type TimeString struct {
	time.Time
}

// Scan implements [database/sql.Scanner].
func (t *TimeString) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		t.Time = time.Time{}
		return nil
	case string:
		return t.parseString(v)
	case []byte:
		return t.parseString(string(v))
	case time.Time:
		t.Time = v
		return nil
	default:
		return fmt.Errorf("db: cannot scan %T into TimeString", value)
	}
}

func (t *TimeString) parseString(s string) error {
	// Two canonical formats:
	//   - RFC3339Nano: used for all explicit inserts (SQLite stores times as strings).
	//   - time.DateTime: produced by SQLite's DEFAULT current_timestamp.
	// Postgres returns time.Time directly, handled by the type-switch in Scan.
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, time.DateTime} {
		if parsed, err := time.Parse(layout, s); err == nil {
			t.Time = parsed
			return nil
		}
	}
	return fmt.Errorf("db: cannot parse %q as a time value", s)
}

// NullTimeScanner implements [database/sql.Scanner] for nullable time.Time values
// stored as RFC3339 strings. T must point to the *time.Time field on
// the destination struct; it is set to nil when the column value is NULL.
type NullTimeScanner struct {
	T **time.Time
}

// Scan implements [database/sql.Scanner].
func (n NullTimeScanner) Scan(value any) error {
	if value == nil {
		*n.T = nil
		return nil
	}
	var ts TimeString
	if err := ts.Scan(value); err != nil {
		return err
	}
	t := ts.Time
	*n.T = &t
	return nil
}
