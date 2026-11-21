package krbctl

import (
	"bytes"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// goldenSuperUserSecret stands in for the random superuser secret in golden krb.json files.
const goldenSuperUserSecret = "generated-superuser-secret"

// superUserSecretPattern matches the clientSecret line krbctl writes: 43 URL-safe characters.
var superUserSecretPattern = regexp.MustCompile(`"clientSecret": "([A-Za-z0-9_-]{43})"`)

// superUserSecret returns the generated superuser secret in the krb.json at path.
func superUserSecret(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated file %s: %v", path, err)
	}

	m := superUserSecretPattern.FindSubmatch(data)
	if m == nil {
		t.Fatalf("%s has no generated superuser secret\n%s", path, data)
	}

	return string(m[1])
}

// replaceSuperUserSecret swaps the random superuser secret in the krb.json at path for
// goldenSuperUserSecret, so the file can be compared with a golden fixture.
func replaceSuperUserSecret(t *testing.T, path string) {
	t.Helper()

	secret := superUserSecret(t, path)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated file %s: %v", path, err)
	}

	data = bytes.Replace(data, []byte(secret), []byte(goldenSuperUserSecret), 1)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write generated file %s: %v", path, err)
	}
}

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

// assertFileNotContains fails the test if the file at path contains unwanted.
func assertFileNotContains(t *testing.T, path, unwanted string) {
	t.Helper()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated file %s: %v", path, err)
	}

	if strings.Contains(string(got), unwanted) {
		t.Errorf("%s contains %q\n%s", path, unwanted, got)
	}
}
