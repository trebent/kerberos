//nolint:gochecknoglobals,cyclop // Package env provides environment variable parsing for the Kerberos service.
package env

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

// ValidateDirPath validates that the given string is a valid directory path.
var ValidateDirPath = func(path string) error {
	_, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	return nil
}

// ValidatePort validates that the given integer is a valid port number (between 1000 and 65535).
var ValidatePort = func(v int) error {
	if v < 1000 || v > 65535 {
		return fmt.Errorf("must be between 1000 and 65535: %d", v)
	}
	return nil
}

// ValidateGreaterThanZero validates that the given integer is greater than zero.
var ValidateGreaterThanZero = func(v int) error {
	if v < 1 {
		return fmt.Errorf("must be greater than 0: %d", v)
	}
	return nil
}

// ValidateGreaterThanOrEqualToZero validates that the given integer is greater than or equal to zero.
var ValidateGreaterThanOrEqualToZero = func(v int) error {
	if v < 0 {
		return fmt.Errorf("must be greater than or equal to 0: %d", v)
	}
	return nil
}

var ValidateHost = func(host string) error {
	if host == "" {
		return errors.New("host cannot be empty")
	}

	h, portStr, err := net.SplitHostPort(host)
	if err == nil {
		// host:port format — validate port then fall through to validate host part.
		port, convErr := strconv.Atoi(portStr)
		if convErr != nil {
			return fmt.Errorf("invalid port %q: %w", portStr, convErr)
		}

		if portErr := ValidatePort(port); portErr != nil {
			return fmt.Errorf("invalid port: %w", portErr)
		}

		host = h
	}
	// If SplitHostPort failed, treat the entire string as a host-only value.

	if net.ParseIP(host) != nil {
		return nil
	}

	return ValidateHostname(host)
}

// ValidateHostname reports whether s is a syntactically valid DNS hostname.
// It enforces RFC 1123: dot-separated labels, each 1–63 characters of
// [A-Za-z0-9] with interior hyphens permitted, total length ≤ 253.
func ValidateHostname(s string) error {
	if n := len(s); n == 0 || n > 253 {
		return fmt.Errorf("hostname had length %d, must be between 1 and 253 characters", len(s))
	}

	for label := range strings.SplitSeq(s, ".") {
		n := len(label)
		if n == 0 || n > 63 {
			return fmt.Errorf(
				"hostname segment had length %d, must be between 1 and 63 characters", n,
			)
		}

		for i, c := range label {
			switch {
			case c >= 'a' && c <= 'z':
			case c >= 'A' && c <= 'Z':
			case c >= '0' && c <= '9':
			case c == '-' && i > 0 && i < n-1:
			default:
				return errors.New("hostname segment contained invalid character: " + string(c))
			}
		}
	}

	return nil
}
