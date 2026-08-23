package env_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/trebent/kerberos/internal/env"
)

func TestValidateDirPath(t *testing.T) {
	t.Parallel()

	existing := t.TempDir()
	missing := filepath.Join(existing, "does-not-exist")

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid temp dir", existing, false},
		{"non-existent path", missing, true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := env.ValidateDirPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDirPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestValidateDirPathFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	f, err := os.CreateTemp(dir, "testfile")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	// A file path is not a directory — ReadDir should fail.
	if err := env.ValidateDirPath(f.Name()); err == nil {
		t.Errorf("ValidateDirPath(%q) expected error for a file path, got nil", f.Name())
	}
}

func TestValidatePort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"minimum valid port", 1000, false},
		{"maximum valid port", 65535, false},
		{"mid-range port", 8080, false},
		{"below minimum", 999, true},
		{"zero", 0, true},
		{"negative", -1, true},
		{"above maximum", 65536, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := env.ValidatePort(tt.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePort(%d) error = %v, wantErr %v", tt.port, err, tt.wantErr)
			}
		})
	}
}

func TestValidateGreaterThanZero(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{"one", 1, false},
		{"large positive", 1000000, false},
		{"zero", 0, true},
		{"negative", -5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := env.ValidateGreaterThanZero(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateGreaterThanZero(%d) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestValidateGreaterThanOrEqualToZero(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   int
		wantErr bool
	}{
		{"zero", 0, false},
		{"positive", 42, false},
		{"large positive", 1000000, false},
		{"negative one", -1, true},
		{"large negative", -100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := env.ValidateGreaterThanOrEqualToZero(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateGreaterThanOrEqualToZero(%d) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestValidateHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		host    string
		wantErr bool
	}{
		// Plain hostnames
		{"simple hostname", "localhost", false},
		{"dotted hostname", "example.com", false},
		{"subdomain", "api.example.com", false},
		{"hostname with hyphens", "my-host.example.com", false},
		{"uppercase hostname", "MyHost.Example.COM", false},

		// Plain IP addresses
		{"IPv4 address", "192.168.1.1", false},
		{"loopback IPv4", "127.0.0.1", false},
		{"IPv6 address", "::1", false},

		// host:port combinations
		{"hostname with valid port", "localhost:8080", false},
		{"dotted hostname with port", "example.com:30000", false},
		{"IPv4 with port", "192.168.1.1:5432", false},
		{"IPv6 with port", "[::1]:8080", false},

		// Invalid inputs
		{"empty string", "", true},
		{"port below minimum", "localhost:999", true},
		{"port above maximum", "localhost:65536", true},
		{"port zero", "localhost:0", true},
		{"non-numeric port", "localhost:abc", true},
		{"leading hyphen in label", "-bad.example.com", true},
		{"trailing hyphen in label", "bad-.example.com", true},
		{"empty label (double dot)", "bad..example.com", true},
		{"invalid character", "bad_host.example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := env.ValidateHost(tt.host)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHost(%q) error = %v, wantErr %v", tt.host, err, tt.wantErr)
			}
		})
	}
}
