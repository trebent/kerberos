package krbctl

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestConfigNonInteractive generates a full Kerberos configuration (echo backend
// + one auth-enabled backend + postgres + observability stack + admin-connector)
// in non-interactive mode and validates the dynamically-built config files
// against golden fixtures. Embedded static assets (grafana dashboards, jaeger
// config) are intentionally not compared.
func TestConfigNonInteractive(t *testing.T) {
	dir := t.TempDir()

	runKrbctl(t, "config", "-y",
		"--echo",
		"--backend", "name=api,host=api,port=8080,auth=true",
		"--driver", "postgres",
		"--obs-stack",
		"--connector",
		"-o", dir)

	files := []string{
		"krb.json",
		"connector.json",
		"prometheus.yml",
		filepath.Join("grafana", "grafana.ini"),
		filepath.Join("grafana", "grafana-datasources.yml"),
	}

	for _, f := range files {
		assertGoldenFile(t,
			filepath.Join(dir, f),
			filepath.Join("testdata", "config", f))
	}
}

// TestConfigInputErrors checks that bad backend input is rejected instead of silently replaced.
func TestConfigInputErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "duplicate backend names",
			args: []string{"--backend", "name=api,host=a", "--backend", "name=api,host=b"},
			want: `duplicate backend name: "api"`,
		},
		{
			name: "echo backend and backend named echo",
			args: []string{"--echo", "--backend", "name=echo"},
			want: `duplicate backend name: "echo"`,
		},
		{
			name: "port not a number",
			args: []string{"--backend", "name=api,port=abc"},
			want: "port must be a number between 1 and 65535",
		},
		{
			name: "port out of range",
			args: []string{"--backend", "name=api,port=70000"},
			want: "port must be a number between 1 and 65535",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"config", "-y", "-o", t.TempDir()}, tt.args...)

			if out := runKrbctlFailure(t, args...); !strings.Contains(out, tt.want) {
				t.Errorf("output does not contain %q:\n%s", tt.want, out)
			}
		})
	}
}

// TestConfigConnectorMessage checks that krbctl reports where connector.json was written.
func TestConfigConnectorMessage(t *testing.T) {
	dir := t.TempDir()

	out := runKrbctlOutput(t, "config", "-y", "--echo", "--connector", "-o", dir)

	if want := "connector.json written to " + dir; !strings.Contains(out, want) {
		t.Errorf("output does not contain %q:\n%s", want, out)
	}
}
