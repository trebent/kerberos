package krbctl

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestValidate checks that krbctl validate accepts a valid config of each kind and rejects a
// config of the other kind.
func TestValidate(t *testing.T) {
	krb := filepath.Join("testdata", "config", "krb.json")
	connector := filepath.Join("testdata", "config", "connector.json")

	t.Run("kerberos ok", func(t *testing.T) {
		assertOutputContains(t, runKrbctlOutput(t, "validate", "kerberos", krb), krb+": ok")
	})

	t.Run("kerberos invalid", func(t *testing.T) {
		assertOutputContains(t, runKrbctlFailure(t, "val", "krb", connector), "gateway is required")
	})

	t.Run("connector ok", func(t *testing.T) {
		assertOutputContains(t, runKrbctlOutput(t, "validate", "connector", connector), connector+": ok")
	})

	t.Run("connector invalid", func(t *testing.T) {
		assertOutputContains(t, runKrbctlFailure(t, "val", "conn", krb), "Additional property gateway")
	})
}

// assertOutputContains fails the test unless out contains want.
func assertOutputContains(t *testing.T, out, want string) {
	t.Helper()

	if !strings.Contains(out, want) {
		t.Errorf("output does not contain %q\n%s", want, out)
	}
}
