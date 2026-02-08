package util

import "testing"

func TestPassword(t *testing.T) {
	_, salt, hash := MakePassword("123")
	if !PasswordMatch(salt, hash, "123") {
		t.Fatal("Should have matched...")
	}

	// Force random
	pw, salt, hash := MakePassword("")
	if !PasswordMatch(salt, hash, pw) {
		t.Fatal("Should have matched...")
	}
}
