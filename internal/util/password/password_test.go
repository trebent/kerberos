package password

import "testing"

func TestPassword(t *testing.T) {
	_, salt, hash := Make("123")
	if !Match(salt, hash, "123") {
		t.Fatal("Should have matched...")
	}

	// Force random
	pw, salt, hash := Make("")
	if !Match(salt, hash, pw) {
		t.Fatal("Should have matched...")
	}
}
