package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
)

//go:embed assets/grafana/prometheus.json
var grafanaDashboardPrometheus []byte

//go:embed assets/grafana/kerberos_runtime.json
var grafanaDashboardRuntime []byte

//go:embed assets/grafana/kerberos_http.json
var grafanaDashboardHTTP []byte

// scrapeTargetPorts maps each known scrape target to its default host:port.
var scrapeTargetPorts = map[string]string{
	defaultKRBDB: defaultKRBDB + ":9464",
	"echo":       "echo:9463",
	"connector":  "connector:9462",
	"jaeger":     "jaeger:8888",
}

// writeObsFiles creates all observability config files in the current directory.
func writeObsFiles(opts *configOptions) error {
	prometheusData := []byte(buildPrometheusYML(&opts.obsOpts))
	if err := writeObsFile("prometheus.yml", prometheusData); err != nil {
		return err
	}

	if err := os.MkdirAll("grafana", 0o750); err != nil {
		return fmt.Errorf("failed to create grafana directory: %w", err)
	}

	grafanaFiles := []struct {
		path string
		data []byte
	}{
		{"grafana/grafana.ini", []byte(buildGrafanaINI(&opts.obsOpts))},
		{"grafana/grafana-datasources.yml", []byte(buildGrafanaDatasourcesYML())},
		{"grafana/grafana-dashboards.yml", []byte(buildGrafanaDashboardsYML())},
		{"grafana/prometheus.json", grafanaDashboardPrometheus},
		{"grafana/kerberos_runtime.json", grafanaDashboardRuntime},
		{"grafana/kerberos_http.json", grafanaDashboardHTTP},
	}

	for _, f := range grafanaFiles {
		if err := writeObsFile(f.path, f.data); err != nil {
			return err
		}
	}

	return writeObsFile("jaeger.yml", []byte(buildJaegerYML()))
}

// writeObsFile writes data to path and prints a confirmation line.
func writeObsFile(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}

	fmt.Fprintf(os.Stdout, "%s written\n", path)

	return nil
}

// buildPrometheusYML generates a prometheus.yml with scrape configs for the selected targets.
func buildPrometheusYML(opts *obsConfigOptions) string {
	var b strings.Builder

	b.WriteString("global:\n  scrape_interval: 15s\n\nscrape_configs:\n")

	for _, target := range opts.scrapeTargets {
		hostPort, ok := scrapeTargetPorts[target]
		if !ok {
			continue
		}

		fmt.Fprintf(&b,
			"  - job_name: %s\n    static_configs:\n      - targets: [\"%s\"]\n",
			target, hostPort,
		)
	}

	return b.String()
}

// buildGrafanaINI generates a slim grafana.ini with only the sections used in this deployment.
func buildGrafanaINI(opts *obsConfigOptions) string {
	var b strings.Builder

	b.WriteString("[database]\n")

	if opts.grafanaDB == driverPostgres {
		b.WriteString("type = postgres\n")
		b.WriteString("host = postgres:5432\n")
		b.WriteString("name = kerberos\n")
		b.WriteString("user = kerberos\n")
		b.WriteString("password = kerberos\n")
	} else {
		b.WriteString("type = sqlite3\n")
	}

	b.WriteString("\n[auth.anonymous]\n")

	if opts.grafanaAnonymous {
		b.WriteString("enabled = true\n")
		b.WriteString("org_name = Main Org.\n")
		b.WriteString("org_role = Viewer\n")
	} else {
		b.WriteString("enabled = false\n")
	}

	return b.String()
}

// buildGrafanaDatasourcesYML generates the Grafana datasource provisioning file.
func buildGrafanaDatasourcesYML() string {
	return `apiVersion: 1

datasources:
  - name: prometheus
    type: prometheus
    url: http://prometheus:9090
    uid: prometheus
`
}

// buildGrafanaDashboardsYML generates the Grafana dashboard provisioning file.
func buildGrafanaDashboardsYML() string {
	return `apiVersion: 1

providers:
  - name: "Prometheus status"
    type: file
    options:
      path: /var/lib/grafana/prometheus.json
  - name: "Kerberos runtime"
    type: file
    options:
      path: /var/lib/grafana/kerberos_runtime.json
  - name: "Kerberos HTTP"
    type: file
    options:
      path: /var/lib/grafana/kerberos_http.json
`
}

// buildJaegerYML returns the static Jaeger configuration.
func buildJaegerYML() string {
	return `service:
  extensions: [jaeger_storage, jaeger_query]
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [jaeger_storage_exporter]
  telemetry:
    resource:
      service.name: jaeger
    metrics:
      level: detailed
      readers:
        - pull:
            exporter:
              prometheus:
                host: 0.0.0.0
                port: 8888
    logs:
      level: info

extensions:
  jaeger_query:
    storage:
      traces: badger_store
      traces_archive: badger_archive
    ui:
      config_file: /config-ui.json
    http:
      endpoint: 0.0.0.0:16686
    grpc:
      endpoint: 0.0.0.0:16685
  jaeger_storage:
    backends:
      badger_store:
        badger:
          directories:
            keys: "/jaeger/"
            values: "/jaeger/"
          ephemeral: false
          ttl:
            spans: 48h
          metrics_update_interval: 10s
      badger_archive:
        badger:
          directories:
            keys: "/jaeger/archive/"
            values: "/jaeger/archive/"
          ephemeral: false
          ttl:
            spans: 72h
          metrics_update_interval: 10s

receivers:
  otlp:
    protocols:
      grpc:
        endpoint: "0.0.0.0:4317"

processors:
  batch:

exporters:
  jaeger_storage_exporter:
    trace_storage: badger_store
`
}
