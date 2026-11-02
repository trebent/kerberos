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
	// the dashboard does not use Prometheus.
	target string
	// service is the observability stack service the dashboard reads its data from.
	service string
}

var dashboards = []dashboard{
	{"prometheus.json", grafanaDashboardPrometheus, anyScrapeTarget, prometheusName},
	{"kerberos_runtime.json", grafanaDashboardRuntime, defaultKRBDB, prometheusName},
	{"kerberos_http.json", grafanaDashboardHTTP, defaultKRBDB, prometheusName},
	{"jaeger.json", grafanaDashboardJaeger, jaegerName, prometheusName},
	{"loki.json", grafanaDashboardLoki, lokiName, prometheusName},
	{"service_logs.json", grafanaDashboardServiceLogs, "", lokiName},
}

// included reports whether the dashboard has data to show given the chosen services and the
// scraped job names.
func (d dashboard) included(sel *obsSelection, scraped map[string]bool) bool {
	if !sel.includes(d.service) {
		return false
	}

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

// obsFile is a generated observability config file.
type obsFile struct {
	path string
	data []byte
}

// writeObsFiles creates the config files of the chosen observability stack services in the
// output directory.
func writeObsFiles(driver string, opts *configOptions) error {
	var files []obsFile

	if opts.includes(prometheusName) {
		files = append(files, obsFile{
			filepath.Join(opts.outputPath, "prometheus.yml"),
			[]byte(buildPrometheusYML(resolveScrapeJobs(opts))),
		})
	}

	if opts.includes(jaegerName) {
		files = append(files,
			obsFile{filepath.Join(opts.outputPath, "jaeger.yml"), jaegerYML},
			obsFile{filepath.Join(opts.outputPath, "jaeger-config-ui.json"), jaegerConfigUI},
		)
	}

	if opts.includes(lokiName) {
		files = append(files, obsFile{filepath.Join(opts.outputPath, "loki.yml"), lokiYML})
	}

	for _, f := range files {
		if err := writeObsFile(f.path, f.data); err != nil {
			return err
		}
	}

	if opts.includes(grafanaName) {
		return writeGrafanaFiles(driver, opts)
	}

	return nil
}

// writeGrafanaFiles creates the Grafana config files and dashboards in the grafana directory.
func writeGrafanaFiles(driver string, opts *configOptions) error {
	grafanaDir := filepath.Join(opts.outputPath, "grafana")
	if err := os.MkdirAll(grafanaDir, 0o750); err != nil {
		return fmt.Errorf("failed to create grafana directory: %w", err)
	}

	grafanaFiles := []obsFile{
		{filepath.Join(grafanaDir, "grafana.ini"), []byte(buildGrafanaINI(driver, &opts.obsOpts))},
		{
			filepath.Join(grafanaDir, "grafana-datasources.yml"),
			[]byte(buildGrafanaDatasourcesYML(&opts.obsSelection)),
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

	return writeDashboards(filepath.Join(grafanaDir, "dashboards"), opts)
}

// writeDashboards writes the Grafana dashboards that have data to show for the chosen services
// and scrape jobs into dir, and removes the other dashboards in case an earlier run wrote them
// there.
func writeDashboards(dir string, opts *configOptions) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("failed to create grafana dashboards directory: %w", err)
	}

	jobs := resolveScrapeJobs(opts)

	scraped := make(map[string]bool, len(jobs))
	for _, job := range jobs {
		scraped[job.name] = true
	}

	for _, d := range dashboards {
		path := filepath.Join(dir, d.file)

		if d.included(&opts.obsSelection, scraped) {
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
// scraped on the default metrics port. Jaeger, Loki and the admin-connector are skipped
// when they are not included in the deployment, and there are no jobs without Prometheus.
//
//nolint:cyclop // one case per kind of scrape target
func resolveScrapeJobs(opts *configOptions) []scrapeJob {
	if !opts.includes(prometheusName) {
		return nil
	}

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
			if opts.includes(jaegerName) {
				jobs = append(jobs, scrapeJob{jaegerName, "jaeger:8888"})
			}
		case lokiName:
			if opts.includes(lokiName) {
				jobs = append(jobs, scrapeJob{lokiName, "loki:3100"})
			}
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

// grafanaDatasources maps each observability stack service Grafana can read from to its URL.
var grafanaDatasources = []struct {
	service string
	url     string
}{
	{prometheusName, "http://prometheus:9090"},
	{lokiName, "http://loki:3100"},
}

// buildGrafanaDatasourcesYML generates the Grafana data sources of the chosen services.
func buildGrafanaDatasourcesYML(sel *obsSelection) string {
	var b strings.Builder

	b.WriteString("apiVersion: 1\n\ndatasources:")

	written := false

	for _, ds := range grafanaDatasources {
		if !sel.includes(ds.service) {
			continue
		}

		fmt.Fprintf(&b, "\n  - name: %[1]s\n    type: %[1]s\n    url: %[2]s\n    uid: %[1]s",
			ds.service, ds.url)

		written = true
	}

	if !written {
		b.WriteString(" []")
	}

	b.WriteString("\n")

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
