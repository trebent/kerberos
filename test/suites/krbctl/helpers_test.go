package krbctl

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// runKrbctl runs the krbctl binary with the given args, failing the test on a
// non-zero exit and surfacing the combined output for diagnostics.
func runKrbctl(t *testing.T, args ...string) {
	t.Helper()

	out, err := exec.Command(krbctlBin, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("krbctl %v failed: %v\n%s", args, err, out)
	}
}

// runKrbctlOutput runs the krbctl binary like runKrbctl and returns its combined output.
func runKrbctlOutput(t *testing.T, args ...string) string {
	t.Helper()

	out, err := exec.Command(krbctlBin, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("krbctl %v failed: %v\n%s", args, err, out)
	}

	return string(out)
}

// runKrbctlFailure runs the krbctl binary, expecting a non-zero exit, and returns its combined
// output.
func runKrbctlFailure(t *testing.T, args ...string) string {
	t.Helper()

	out, err := exec.Command(krbctlBin, args...).CombinedOutput()
	if err == nil {
		t.Fatalf("krbctl %v succeeded, expected a failure\n%s", args, out)
	}

	return string(out)
}

// assertGoldenFile compares the file at genPath byte-for-byte against the golden
// fixture at goldenPath.
func assertGoldenFile(t *testing.T, genPath, goldenPath string) {
	t.Helper()

	got, err := os.ReadFile(genPath)
	if err != nil {
		t.Fatalf("read generated file %s: %v", genPath, err)
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden file %s: %v", goldenPath, err)
	}

	if !bytes.Equal(got, want) {
		t.Errorf("generated %s does not match golden %s\n--- got ---\n%s\n--- want ---\n%s",
			genPath, goldenPath, got, want)
	}
}

// assertFileContains fails the test unless the file at path contains want.
func assertFileContains(t *testing.T, path, want string) {
	t.Helper()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated file %s: %v", path, err)
	}

	if !strings.Contains(string(got), want) {
		t.Errorf("%s does not contain %q\n%s", path, want, got)
	}
}
