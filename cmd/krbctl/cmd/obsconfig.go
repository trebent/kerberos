package cmd

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
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

//go:embed assets/grafana/jaeger.json
var grafanaDashboardJaeger []byte

//go:embed assets/grafana/loki.json
var grafanaDashboardLoki []byte

//go:embed assets/grafana/service_logs.json
var grafanaDashboardServiceLogs []byte

//go:embed assets/grafana/grafana-datasources.yml
var grafanaDataSourceYML []byte

//go:embed assets/grafana/grafana-dashboards.yml
var grafanaDashboardsYML []byte

//go:embed assets/jaeger/jaeger-config-ui.json
var jaegerConfigUI []byte

//go:embed assets/jaeger/jaeger.yml
var jaegerYML []byte

//go:embed assets/loki/loki.yml
var lokiYML []byte

// anyScrapeTarget marks a dashboard that is included when at least one scrape job exists.
const anyScrapeTarget = "*"

// dashboard is a provisioned Grafana dashboard and the scrape job it reads its metrics from.
type dashboard struct {
	file string
	data []byte
	// target is the scrape job the dashboard needs, anyScrapeTarget for any job, or empty when
	// the dashboard does not use Prometheus and is always included.
	target string
}

var dashboards = []dashboard{
	{"prometheus.json", grafanaDashboardPrometheus, anyScrapeTarget},
	{"kerberos_runtime.json", grafanaDashboardRuntime, defaultKRBDB},
	{"kerberos_http.json", grafanaDashboardHTTP, defaultKRBDB},
	{"jaeger.json", grafanaDashboardJaeger, jaegerName},
	{"loki.json", grafanaDashboardLoki, lokiName},
	{"service_logs.json", grafanaDashboardServiceLogs, ""},
}

// included reports whether the dashboard has data to show given the scraped job names.
func (d dashboard) included(scraped map[string]bool) bool {
	switch d.target {
	case "":
		return true
	case anyScrapeTarget:
		return len(scraped) > 0
	default:
		return scraped[d.target]
	}
}

// scrapeJob is a resolved Prometheus scrape target (job name + host:port).
type scrapeJob struct {
	name   string
	target string
}

// writeObsFiles creates all observability config files in the current directory.
func writeObsFiles(driver string, opts *configOptions) error {
	jobs := resolveScrapeJobs(opts)

	prometheusData := []byte(buildPrometheusYML(jobs))
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
	}

	for _, f := range grafanaFiles {
		if err := writeObsFile(f.path, f.data); err != nil {
			return err
		}
	}

	if err := writeDashboards(filepath.Join(grafanaDir, "dashboards"), jobs); err != nil {
		return err
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

	if err := writeObsFile(filepath.Join(opts.outputPath, "loki.yml"), lokiYML); err != nil {
		return err
	}

	return nil
}

// writeDashboards writes the Grafana dashboards that have data to show for the given scrape
// jobs into dir, and removes the other dashboards in case an earlier run wrote them there.
func writeDashboards(dir string, jobs []scrapeJob) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("failed to create grafana dashboards directory: %w", err)
	}

	scraped := make(map[string]bool, len(jobs))
	for _, job := range jobs {
		scraped[job.name] = true
	}

	for _, d := range dashboards {
		path := filepath.Join(dir, d.file)

		if d.included(scraped) {
			if err := writeObsFile(path, d.data); err != nil {
				return err
			}

			continue
		}

		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
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
// jobs. kerberos, jaeger, loki and the admin-connector resolve to their well-known
// endpoints, every other selected target is treated as a registered router backend
// scraped on the default metrics port. The admin-connector is skipped when it is not
// included in the deployment.
func resolveScrapeJobs(opts *configOptions) []scrapeJob {
	hosts := make(map[string]string, len(opts.backends))
	for _, b := range opts.backends {
		hosts[b.name] = b.host
	}

	jobs := make([]scrapeJob, 0, len(opts.obsOpts.scrapeTargets))

	for _, target := range opts.obsOpts.scrapeTargets {
		switch target {
		case defaultKRBDB:
			jobs = append(
				jobs,
				scrapeJob{defaultKRBDB, fmt.Sprintf("%s:%d", defaultKRBDB, scrapeMetricsPort)},
			)
		case jaegerName:
			jobs = append(jobs, scrapeJob{jaegerName, "jaeger:8888"})
		case lokiName:
			jobs = append(jobs, scrapeJob{lokiName, "loki:3100"})
		case connectorName:
			if opts.includeConnector {
				target := fmt.Sprintf("%s:%d", connectorName, scrapeMetricsPort)
				jobs = append(jobs, scrapeJob{connectorName, target})
			}
		default:
			host, ok := hosts[target]
			if !ok {
				continue
			}

			jobs = append(jobs, scrapeJob{target, fmt.Sprintf("%s:%d", host, scrapeMetricsPort)})
		}
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
