package password

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	saltBytes = 32
	// bcryptMaxInput is the largest input bcrypt accepts. Longer inputs are rejected.
	bcryptMaxInput = 72
)

// Make creates a random password if the input password is "". It returns the input or generated
// password, the salt, and the hashed version of the password.
func Make(password string) (string, string, string, error) {
	if password == "" {
		password = uuid.NewString()
	}

	salt := make([]byte, saltBytes)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", "", "", fmt.Errorf("generate salt: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword(hashInput(salt, password), bcrypt.DefaultCost)
	if err != nil {
		return "", "", "", fmt.Errorf("hash password: %w", err)
	}

	return password, hex.EncodeToString(salt), hex.EncodeToString(hash), nil
}

// Match verifies if the input cleartext password matches the salt and hashed password.
func Match(salt, hashedPassword, clearTextPassword string) bool {
	decodedSalt, err := hex.DecodeString(salt)
	if err != nil {
		return false
	}

	hashedPasswordBytes, err := hex.DecodeString(hashedPassword)
	if err != nil {
		return false
	}

	return bcrypt.CompareHashAndPassword(
		hashedPasswordBytes,
		hashInput(decodedSalt, clearTextPassword),
	) == nil
}

// hashInput builds the bcrypt input from the salt and the password's UTF-8 bytes. When the result
// would exceed bcrypt's input limit, the password is replaced by its SHA-256 digest so the input
// always fits. Shorter inputs are used as they are, which keeps existing hashes valid.
func hashInput(salt []byte, password string) []byte {
	pw := []byte(password)
	if len(salt)+len(pw) > bcryptMaxInput {
		digest := sha256.Sum256(pw)
		pw = digest[:]
	}

	input := make([]byte, 0, len(salt)+len(pw))
	input = append(input, salt...)

	return append(input, pw...)
}
