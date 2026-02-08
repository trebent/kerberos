package util

import (
	"crypto/rand"
	"encoding/hex"
	"io"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const saltBytes = 32

// MakePassword creates a random password if the input password is "". It will return the input/generated
// password, the salt, and the hashed version of the password.
func MakePassword(password string) (string, string, string) {
	if password == "" {
		password = uuid.NewString()
	}

	salt := make([]byte, saltBytes)
	_, err := io.ReadFull(rand.Reader, salt)
	if err != nil {
		panic(err)
	}

	fullPassword := make([]byte, saltBytes+len(password))
	copy(fullPassword, salt)
	for i, c := range password {
		fullPassword[len(salt)+i] = byte(c)
	}
	hash, err := bcrypt.GenerateFromPassword(fullPassword, bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	return password, hex.EncodeToString(salt), hex.EncodeToString(hash)
}

// PasswordMatch verifies if the input cleartext password matches the salt and hashed password.
func PasswordMatch(salt, hashedPassword, clearTextPassword string) bool {
	decodedSalt, _ := hex.DecodeString(salt)
	hashedPasswordBytes, _ := hex.DecodeString(hashedPassword)
	return bcrypt.CompareHashAndPassword(
		hashedPasswordBytes,
		append(decodedSalt, []byte(clearTextPassword)...),
	) == nil
}
