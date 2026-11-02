package krbctl

import (
	"os"
	"path/filepath"
	"slices"
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

// TestConfigKerberosAndEcho generates the observability stack with only kerberos and echo as
// scrape targets and validates prometheus.yml against the golden fixture, along with the
// dashboards that are written for those targets.
func TestConfigKerberosAndEcho(t *testing.T) {
	dir := t.TempDir()

	runKrbctl(t, "config", "-y",
		"--echo",
		"--obs-stack",
		"--scrape-target", "kerberos",
		"--scrape-target", "echo",
		"-o", dir)

	assertGoldenFile(t,
		filepath.Join(dir, "prometheus.yml"),
		filepath.Join("testdata", "config-kerberos-echo", "prometheus.yml"))

	assertDashboards(t, filepath.Join(dir, "grafana", "dashboards"), []string{
		"prometheus.json",
		"kerberos_runtime.json",
		"kerberos_http.json",
		"service_logs.json",
	})
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

// TestConfigDashboards checks that only the Grafana dashboards matching the selected scrape
// targets are written, and that a later run removes dashboards that no longer apply.
func TestConfigDashboards(t *testing.T) {
	tests := []struct {
		name    string
		targets []string
		want    []string
	}{
		{
			name: "default targets",
			want: allDashboards,
		},
		{
			name:    "kerberos only",
			targets: []string{"kerberos"},
			want:    []string{"prometheus.json", "kerberos_runtime.json", "kerberos_http.json", "service_logs.json"},
		},
		{
			name:    "jaeger only",
			targets: []string{"jaeger"},
			want:    []string{"prometheus.json", "jaeger.json", "service_logs.json"},
		},
		{
			name:    "backend only",
			targets: []string{"echo"},
			want:    []string{"prometheus.json", "service_logs.json"},
		},
	}

	dir := t.TempDir()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"config", "-y", "--echo", "--obs-stack", "-o", dir}
			for _, target := range tt.targets {
				args = append(args, "--scrape-target", target)
			}

			runKrbctl(t, args...)

			assertDashboards(t, filepath.Join(dir, "grafana", "dashboards"), tt.want)
		})
	}
}

// allDashboards lists every Grafana dashboard krbctl can generate.
var allDashboards = []string{
	"prometheus.json",
	"kerberos_runtime.json",
	"kerberos_http.json",
	"jaeger.json",
	"loki.json",
	"service_logs.json",
}

// assertDashboards fails the test unless exactly the dashboards in want exist in dir.
func assertDashboards(t *testing.T, dir string, want []string) {
	t.Helper()

	for _, f := range allDashboards {
		_, err := os.Stat(filepath.Join(dir, f))
		exists := err == nil

		if expected := slices.Contains(want, f); exists != expected {
			t.Errorf("dashboard %s: exists = %t, want %t", f, exists, expected)
		}
	}
}

// TestConfigConnectorScrapeTarget checks that the admin-connector is scraped by default when it
// is included, and only when listed once scrape targets are passed explicitly.
func TestConfigConnectorScrapeTarget(t *testing.T) {
	const job = "job_name: connector"

	tests := []struct {
		name    string
		args    []string
		scraped bool
	}{
		{name: "default targets", args: []string{"--connector"}, scraped: true},
		{name: "listed", args: []string{"--connector", "--scrape-target", "connector"}, scraped: true},
		{name: "not listed", args: []string{"--connector", "--scrape-target", "kerberos"}},
		{name: "listed without connector", args: []string{"--scrape-target", "connector"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			args := append([]string{"config", "-y", "--echo", "--obs-stack", "-o", dir}, tt.args...)
			runKrbctl(t, args...)

			got, err := os.ReadFile(filepath.Join(dir, "prometheus.yml"))
			if err != nil {
				t.Fatalf("read prometheus.yml: %v", err)
			}

			if scraped := strings.Contains(string(got), job); scraped != tt.scraped {
				t.Errorf("connector scraped = %t, want %t\n%s", scraped, tt.scraped, got)
			}
		})
	}
}
