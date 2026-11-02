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

// TestComposeMinimal generates compose.yaml with nothing opted in, which leaves only Kerberos and
// the job that prepares its SQLite file, and validates it against the golden fixture.
func TestComposeMinimal(t *testing.T) {
	dir := t.TempDir()

	runKrbctl(t, "compose", "-y", "-o", dir)

	assertGoldenFile(t,
		filepath.Join(dir, "compose.yaml"),
		filepath.Join("testdata", "compose-minimal", "compose.yaml"))
}

// TestComposeWithoutLoki generates compose.yaml with only Prometheus and Jaeger from the
// observability stack and validates it against the golden fixture: no Loki service, logging
// anchor or Loki dependencies.
func TestComposeWithoutLoki(t *testing.T) {
	dir := t.TempDir()

	runKrbctl(t, "compose", "-y",
		"--obs-stack",
		"--obs-service", "prometheus",
		"--obs-service", "jaeger",
		"--connector",
		"-o", dir)

	assertGoldenFile(t,
		filepath.Join(dir, "compose.yaml"),
		filepath.Join("testdata", "compose-without-loki", "compose.yaml"))
}

// TestComposeObsServiceErrors checks that bad --obs-service input is rejected.
func TestComposeObsServiceErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "unknown service",
			args: []string{"--obs-stack", "--obs-service", "tempo"},
			want: "unknown observability service",
		},
		{
			name: "without obs-stack",
			args: []string{"--obs-service", "loki"},
			want: "--obs-service requires --obs-stack",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"compose", "-y", "-o", t.TempDir()}, tt.args...)

			if out := runKrbctlFailure(t, args...); !strings.Contains(out, tt.want) {
				t.Errorf("output does not contain %q:\n%s", tt.want, out)
			}
		})
	}
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

	t.Run("required without jaeger", func(t *testing.T) {
		out := runKrbctlFailure(t, "compose", "-y", "--connector",
			"--obs-stack", "--obs-service", "prometheus", "-o", t.TempDir())

		if !strings.Contains(out, "--connector-target is required") {
			t.Errorf("unexpected output:\n%s", out)
		}
	})

	t.Run("custom target publishes the jaeger ui", func(t *testing.T) {
		dir := t.TempDir()

		runKrbctl(t, "compose", "-y", "--connector", "--connector-target", "tool:8080",
			"--obs-stack", "-o", dir)

		assertFileContains(t, filepath.Join(dir, "compose.yaml"), "- 16686:16686")
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
