package krbctl

import (
	"path/filepath"
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
		"--echo-backend",
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
