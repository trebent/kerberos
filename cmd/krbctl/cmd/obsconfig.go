package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed assets/grafana/prometheus.json
var grafanaDashboardPrometheus []byte

//go:embed assets/grafana/kerberos_runtime.json
var grafanaDashboardRuntime []byte

//go:embed assets/grafana/kerberos_http.json
var grafanaDashboardHTTP []byte

//go:embed assets/grafana/grafana-datasources.yml
var grafanaDataSourceYML []byte

//go:embed assets/grafana/grafana-dashboards.yml
var grafanaDashboardsYML []byte

//go:embed assets/jaeger/config-ui.json
var jaegerConfigUI []byte

//go:embed assets/jaeger/jaeger.yml
var jaegerYML []byte

// scrapeJob is a resolved Prometheus scrape target (job name + host:port).
type scrapeJob struct {
	name   string
	target string
}

// writeObsFiles creates all observability config files in the current directory.
func writeObsFiles(driver string, opts *configOptions) error {
	prometheusData := []byte(buildPrometheusYML(resolveScrapeJobs(opts)))
	if err := writeObsFile(
		filepath.Join(opts.outputPath, "prometheus.yml"), prometheusData,
	); err != nil {
		return err
	}

	grafanaDir := filepath.Join(opts.outputPath, "grafana")
	if err := os.MkdirAll(grafanaDir, 0o750); err != nil {
		return fmt.Errorf("failed to create grafana directory: %w", err)
	}

	grafanaFiles := []struct {
		path string
		data []byte
	}{
		{filepath.Join(grafanaDir, "grafana.ini"), []byte(buildGrafanaINI(driver, &opts.obsOpts))},
		{
			filepath.Join(grafanaDir, "grafana-datasources.yml"),
			grafanaDataSourceYML,
		},
		{
			filepath.Join(grafanaDir, "grafana-dashboards.yml"),
			grafanaDashboardsYML,
		},
		{filepath.Join(grafanaDir, "prometheus.json"), grafanaDashboardPrometheus},
		{filepath.Join(grafanaDir, "kerberos_runtime.json"), grafanaDashboardRuntime},
		{filepath.Join(grafanaDir, "kerberos_http.json"), grafanaDashboardHTTP},
	}

	for _, f := range grafanaFiles {
		if err := writeObsFile(f.path, f.data); err != nil {
			return err
		}
	}

	if err := writeObsFile(
		filepath.Join(opts.outputPath, "jaeger.yml"), jaegerYML,
	); err != nil {
		return err
	}

	if err := writeObsFile(
		filepath.Join(opts.outputPath, "jaeger-config-ui.json"), jaegerConfigUI,
	); err != nil {
		return err
	}

	return nil
}

// writeObsFile writes data to path and prints a confirmation line.
func writeObsFile(path string, data []byte) error {
	//nolint:gosec // welp
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}

	fmt.Fprintf(os.Stdout, "%s written\n", path)

	return nil
}

// resolveScrapeJobs turns the selected scrape targets into concrete host:port
// jobs. kerberos and jaeger resolve to their well-known endpoints, every other
// selected target is treated as a registered router backend scraped on the
// default metrics port. The admin-connector is added automatically when the
// connector is included in the deployment.
func resolveScrapeJobs(opts *configOptions) []scrapeJob {
	hosts := make(map[string]string, len(opts.backends))
	for _, b := range opts.backends {
		hosts[b.name] = b.host
	}

	jobs := make([]scrapeJob, 0, len(opts.obsOpts.scrapeTargets)+1)

	for _, target := range opts.obsOpts.scrapeTargets {
		switch target {
		case defaultKRBDB:
			jobs = append(
				jobs,
				scrapeJob{defaultKRBDB, fmt.Sprintf("%s:%d", defaultKRBDB, scrapeMetricsPort)},
			)
		case jaegerName:
			jobs = append(jobs, scrapeJob{jaegerName, "jaeger:8888"})
		default:
			host, ok := hosts[target]
			if !ok {
				continue
			}

			jobs = append(jobs, scrapeJob{target, fmt.Sprintf("%s:%d", host, scrapeMetricsPort)})
		}
	}

	if opts.includeConnector {
		jobs = append(jobs, scrapeJob{"connector", fmt.Sprintf("connector:%d", scrapeMetricsPort)})
	}

	return jobs
}

// buildPrometheusYML generates a prometheus.yml with scrape configs for the given jobs.
func buildPrometheusYML(jobs []scrapeJob) string {
	var b strings.Builder

	b.WriteString("global:\n  scrape_interval: 15s\n\nscrape_configs:\n")

	for _, job := range jobs {
		fmt.Fprintf(&b,
			"  - job_name: %s\n    static_configs:\n      - targets: [\"%s\"]\n",
			job.name, job.target,
		)
	}

	return b.String()
}

// buildGrafanaINI generates a slim grafana.ini with only the sections used in this deployment.
func buildGrafanaINI(driver string, opts *obsConfigOptions) string {
	var b strings.Builder

	b.WriteString("[database]\n")

	if driver == driverPostgres {
		b.WriteString("type = postgres\n")
		b.WriteString("host = postgres:5432\n")
		b.WriteString("name = kerberos\n")
		b.WriteString("user = kerberos\n")
		b.WriteString("password = kerberos\n")
	} else {
		b.WriteString("type = sqlite3\n")
		fmt.Fprintf(&b, "path = %s\n", sqliteSharedPath)
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
