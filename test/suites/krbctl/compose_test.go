package krbctl

import (
	"path/filepath"
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
		"--postgres",
		"--connector",
		"-o", dir)

	assertGoldenFile(t,
		filepath.Join(dir, "compose.yaml"),
		filepath.Join("testdata", "compose", "compose.yaml"))
}
