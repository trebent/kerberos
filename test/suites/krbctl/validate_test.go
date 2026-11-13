package krbctl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestValidate checks that krbctl validate accepts generated configs and rejects broken ones,
// both for the Kerberos config and, with -c, the admin-connector config.
func TestValidate(t *testing.T) {
	dir := t.TempDir()

	runKrbctl(t, "config", "-y", "--echo", "--connector", "-o", dir)

	krb := filepath.Join(dir, "krb.json")
	connector := filepath.Join(dir, "connector.json")

	t.Run("kerberos ok", func(t *testing.T) {
		assertOutputContains(t, runKrbctlOutput(t, "validate", krb), krb+": ok")
	})

	t.Run("connector ok", func(t *testing.T) {
		assertOutputContains(t, runKrbctlOutput(t, "v", "-c", connector), connector+": ok")
	})

	t.Run("golden fixtures ok", func(t *testing.T) {
		runKrbctl(t, "v", filepath.Join("testdata", "config", "krb.json"))
		runKrbctl(t, "v", "-c", filepath.Join("testdata", "config", "connector.json"))
	})

	t.Run("wrong kind", func(t *testing.T) {
		assertOutputContains(t, runKrbctlFailure(t, "v", connector), "gateway is required")
		assertOutputContains(t, runKrbctlFailure(t, "v", "-c", krb), "Additional property gateway")
	})

	t.Run("duplicate backend", func(t *testing.T) {
		path := writeFile(t, `{"gateway":{"router":{"backends":[`+
			`{"name":"a","host":"a","port":1},{"name":"a","host":"b","port":2}]}}}`)

		assertOutputContains(t, runKrbctlFailure(t, "v", path), "duplicate backend a")
	})

	t.Run("invalid json", func(t *testing.T) {
		runKrbctlFailure(t, "v", writeFile(t, `{`))
	})

	t.Run("missing file", func(t *testing.T) {
		runKrbctlFailure(t, "v", filepath.Join(dir, "nope.json"))
	})

	t.Run("no usage on failure", func(t *testing.T) {
		out := runKrbctlFailure(t, "v", connector)
		if strings.Contains(out, "Usage:") {
			t.Errorf("validation failure printed usage:\n%s", out)
		}
	})
}

// writeFile writes content to a temporary file and returns its path.
func writeFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "cfg.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}

	return path
}

// assertOutputContains fails the test unless out contains want.
func assertOutputContains(t *testing.T, out, want string) {
	t.Helper()

	if !strings.Contains(out, want) {
		t.Errorf("output does not contain %q\n%s", want, out)
	}
}
