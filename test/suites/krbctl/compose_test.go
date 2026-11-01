package krbctl

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestComposeNonInteractive generates a full compose.yaml (echo + observability
// stack + postgres + admin-connector) in non-interactive mode and validates it
// against the golden fixture.
func TestComposeNonInteractive(t *testing.T) {
	dir := t.TempDir()

	runKrbctl(t, "compose", "-y",
		"--echo",
		"--obs-stack",
		"--driver", "postgres",
		"--connector",
		"-o", dir)

	assertGoldenFile(t,
		filepath.Join(dir, "compose.yaml"),
		filepath.Join("testdata", "compose", "compose.yaml"))
}

// TestComposeConnectorTarget checks that an admin-connector without the observability stack
// needs an explicit target, and that the target ends up in compose.yaml.
func TestComposeConnectorTarget(t *testing.T) {
	t.Run("required without obs-stack", func(t *testing.T) {
		out := runKrbctlFailure(t, "compose", "-y", "--connector", "-o", t.TempDir())

		if !strings.Contains(out, "--connector-target is required") {
			t.Errorf("unexpected output:\n%s", out)
		}
	})

	t.Run("explicit target", func(t *testing.T) {
		dir := t.TempDir()

		runKrbctl(t, "compose", "-y", "--connector", "--connector-target", "tool:8080", "-o", dir)

		assertFileContains(t, filepath.Join(dir, "compose.yaml"), "- TARGET=tool:8080")
	})

	t.Run("target with scheme", func(t *testing.T) {
		out := runKrbctlFailure(t, "compose", "-y", "--connector",
			"--connector-target", "http://tool:8080", "-o", t.TempDir())

		if !strings.Contains(out, "without a scheme or path") {
			t.Errorf("unexpected output:\n%s", out)
		}
	})
}
