package util

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"io"

	"github.com/google/uuid"
)

// MakePassword creates a random password if the input password is "". It will return the input/generated
// password, the salt, and the hashed version of the password.
func MakePassword(password string) (string, string, string) {
	if password == "" {
		password = uuid.NewString()
	}
	hash := sha512.New()

	const saltBytes = 32
	salt := make([]byte, saltBytes)
	_, err := io.ReadFull(rand.Reader, salt)
	if err != nil {
		panic(err)
	}

	_, _ = hash.Write(salt)
	_, _ = hash.Write([]byte(password))
	return password, hex.EncodeToString(salt), hex.EncodeToString(hash.Sum(nil))
}

// PasswordMatch verifies if the input cleartext password matches the salt and hashed password.
func PasswordMatch(salt, hashedPassword, clearTextPassword string) bool {
	decodedSalt, _ := hex.DecodeString(salt)
	hash := sha512.New()
	_, _ = hash.Write(decodedSalt)
	_, _ = hash.Write([]byte(clearTextPassword))
	inputHashed := hex.EncodeToString(hash.Sum(nil))

	return inputHashed == hashedPassword
}
