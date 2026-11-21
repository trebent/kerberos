package password

import (
	"encoding/hex"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPassword(t *testing.T) {
	tests := map[string]string{
		"short":             "123",
		"at bcrypt limit":   strings.Repeat("a", bcryptMaxInput-saltBytes),
		"over bcrypt limit": strings.Repeat("a", 64),
		"non-ASCII":         "lösenord-åäö",
		"long non-ASCII":    strings.Repeat("å", 64),
	}

	for name, pw := range tests {
		t.Run(name, func(t *testing.T) {
			_, salt, hash, err := Make(pw)
			if err != nil {
				t.Fatal(err)
			}

			if !Match(salt, hash, pw) {
				t.Fatal("expected the password to match")
			}

			if Match(salt, hash, pw+"x") {
				t.Fatal("expected a different password not to match")
			}
		})
	}

	t.Run("random", func(t *testing.T) {
		pw, salt, hash, err := Make("")
		if err != nil {
			t.Fatal(err)
		}

		if !Match(salt, hash, pw) {
			t.Fatal("expected the generated password to match")
		}
	})

	t.Run("long passwords differ after 40 bytes", func(t *testing.T) {
		pw := strings.Repeat("a", 64)
		_, salt, hash, err := Make(pw)
		if err != nil {
			t.Fatal(err)
		}

		if Match(salt, hash, pw[:63]+"b") {
			t.Fatal("expected passwords differing only at the end not to match")
		}
	})

	t.Run("hash made before pre-hashing", func(t *testing.T) {
		pw := "legacy-password"
		salt := make([]byte, saltBytes)
		legacy, err := bcrypt.GenerateFromPassword(append(salt, []byte(pw)...), bcrypt.MinCost)
		if err != nil {
			t.Fatal(err)
		}

		if !Match(hex.EncodeToString(salt), hex.EncodeToString(legacy), pw) {
			t.Fatal("expected an existing hash to keep matching")
		}
	})

	t.Run("invalid encoding", func(t *testing.T) {
		if Match("zz", "zz", "123") {
			t.Fatal("expected invalid hex not to match")
		}
	})
}
