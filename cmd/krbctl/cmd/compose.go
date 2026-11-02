package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

// defaultConnectorTarget is what the admin-connector protects when Jaeger is included: the
// Jaeger UI.
const defaultConnectorTarget = "jaeger:16686"

const (
	prometheusName = "prometheus"
	grafanaName    = "grafana"
)

// obsServices lists the services of the observability stack that can be included.
var obsServices = []string{prometheusName, grafanaName, jaegerName, lokiName}

// lokiLoggingAnchor sends container logs to Loki through the Loki Docker logging plugin. The
// plugin runs on the host, so it reaches Loki on the port published on the loopback interface.
// Non-blocking mode and short retries keep services from stalling while Loki is unreachable.
const lokiLoggingAnchor = `x-logging: &loki-logging
  driver: loki
  options:
    loki-url: http://127.0.0.1:3100/loki/api/v1/push
    mode: non-blocking
    max-buffer-size: 4m
    loki-retries: "2"
    loki-max-backoff: 1s
    loki-timeout: 1s

`

// dependency is one depends_on entry of a generated service.
type dependency struct {
	service   string
	condition string
}

// sqliteInitDependency waits for the job that prepares the shared SQLite file.
var sqliteInitDependency = dependency{"sqlite-init", "service_completed_successfully"}

var (
	errConnectorTargetRequired = errors.New(
		"--connector-target is required with --connector when Jaeger is not included")
	errObsServiceWithoutStack = errors.New("--obs-service requires --obs-stack")
	errUnknownObsService      = fmt.Errorf(
		"unknown observability service, must be one of: %s", strings.Join(obsServices, ", "))
	errConnectorTargetFormat = errors.New(
		"connector target must be host:port or host, without a scheme or path")
)

// obsSelection is the choice of observability stack services, shared by the compose and config
// commands.
type obsSelection struct {
	includeObsStack bool

	// obsServices are the observability stack services to include when includeObsStack is set.
	obsServices []string
}

// includes reports whether the given observability stack service is part of the deployment.
func (o *obsSelection) includes(service string) bool {
	return o.includeObsStack && slices.Contains(o.obsServices, service)
}

// composeOptions holds the answers collected from the interactive compose session.
type composeOptions struct {
	outputPath string

	// Enablement flags.
	includeEcho      bool
	includePostgres  bool
	includeConnector bool

	obsSelection

	// connectorTarget is the host:port the admin-connector forwards to (its TARGET).
	connectorTarget string
}

func newComposeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compose",
		Short: "Interactively generate a base compose.yaml for a Kerberos deployment",
		Long: `Walks you through a series of prompts to build a compose.yaml that can
be used to run a Kerberos deployment. Optional sections (observability stack,
postgres, admin-connector, echo) can be included or skipped at each step.`,
		RunE: runCompose,
	}

	cmd.Flags().StringP("output", "o", ".", "Output path where compose.yaml will be written.")
	cmd.Flags().BoolP("non-interactive", "y", false,
		"Skip prompts and build compose.yaml from flag values.")
	cmd.Flags().Bool("echo", false, "Include the echo service. (non-interactive mode only)")
	cmd.Flags().Bool("obs-stack", false,
		"Include the observability stack. (non-interactive mode only)")
	addObsServiceFlag(cmd)
	addDriverFlag(cmd)
	cmd.Flags().Bool("connector", false,
		"Include the admin-connector service. (non-interactive mode only)")
	cmd.Flags().String("connector-target", "",
		"host:port the admin-connector forwards to. Defaults to "+defaultConnectorTarget+
			" when Jaeger is included, required otherwise. (non-interactive mode only)")

	return cmd
}

func runCompose(cmd *cobra.Command, _ []string) error {
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}

	nonInteractive, err := cmd.Flags().GetBool("non-interactive")
	if err != nil {
		return err
	}

	opts := &composeOptions{outputPath: output}

	if nonInteractive {
		if err := collectComposeFromFlags(cmd, opts); err != nil {
			return err
		}
	} else if err := runInteractive(cmd, func() error {
		return collectComposeInteractive(opts)
	}); err != nil {
		return err
	}

	if err := resolveConnectorTarget(opts); err != nil {
		return err
	}

	content := buildCompose(opts)

	//nolint:gosec // welp
	if err := os.WriteFile(
		filepath.Join(opts.outputPath, "compose.yaml"), []byte(content), 0o644,
	); err != nil {
		return fmt.Errorf("failed to write compose file: %w", err)
	}

	fmt.Fprintf(os.Stdout, "\ncompose.yaml written to %s\n", opts.outputPath)

	return nil
}

// collectComposeInteractive drives the interactive prompts, populating opts with the user's
// answers. A single form asks what to include, then follow-up questions dig down into the
// observability stack and the admin-connector.
func collectComposeInteractive(opts *composeOptions) error {
	if err := runForm(huh.NewGroup(
		newConfirm("Include the echo service?", "Useful for testing backends.", &opts.includeEcho),
		newPostgresConfirm(&opts.includePostgres),
		newConfirm("Include the admin-connector service?", "", &opts.includeConnector),
		newConfirm(
			"Include the observability stack?",
			"Choose from Prometheus, Grafana, Jaeger, and Loki next.",
			&opts.includeObsStack,
		),
	)); err != nil {
		return err
	}

	if err := promptObsServices(&opts.obsSelection); err != nil {
		return err
	}

	if opts.includeConnector {
		if opts.includes(jaegerName) {
			opts.connectorTarget = defaultConnectorTarget
		}

		if err := runForm(buildConnectorTargetGroup(opts)); err != nil {
			return err
		}
	}

	return nil
}

// promptObsServices asks which observability stack services to include when the stack is
// included. Every service is selected by default.
func promptObsServices(sel *obsSelection) error {
	if !sel.includeObsStack {
		return nil
	}

	sel.obsServices = slices.Clone(obsServices)

	options := make([]huh.Option[string], 0, len(obsServices))
	for _, service := range obsServices {
		options = append(options, huh.NewOption(service, service))
	}

	return runForm(huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title("Which observability services should be included?").
			Options(options...).
			Validate(func(selected []string) error {
				if len(selected) == 0 {
					return errors.New("select at least one service")
				}

				return nil
			}).
			Value(&sel.obsServices),
	))
}

// buildConnectorTargetGroup asks which tool the admin-connector protects.
func buildConnectorTargetGroup(opts *composeOptions) *huh.Group {
	return huh.NewGroup(
		huh.NewInput().
			Title("Which tool should the admin-connector protect?").
			Description("host:port of the tool, as reachable inside the Compose network.").
			Placeholder("my-tool:8080").
			Validate(validateConnectorTarget).
			Value(&opts.connectorTarget),
	)
}

// collectComposeFromFlags populates opts from the command's flag values for
// non-interactive runs.
func collectComposeFromFlags(cmd *cobra.Command, opts *composeOptions) error {
	var err error

	if opts.includeEcho, err = cmd.Flags().GetBool("echo"); err != nil {
		return err
	}

	if opts.obsSelection, err = getObsSelectionFlags(cmd); err != nil {
		return err
	}

	driver, err := getDriverFlag(cmd)
	if err != nil {
		return err
	}

	opts.includePostgres = driver == driverPostgres

	if opts.includeConnector, err = cmd.Flags().GetBool("connector"); err != nil {
		return err
	}

	if opts.connectorTarget, err = cmd.Flags().GetString("connector-target"); err != nil {
		return err
	}

	return nil
}

// addObsServiceFlag registers the --obs-service flag shared by the compose and config commands.
func addObsServiceFlag(cmd *cobra.Command) {
	cmd.Flags().StringArray("obs-service", nil,
		"Observability stack service to include (repeatable): "+strings.Join(obsServices, ", ")+
			". Defaults to all. Requires --obs-stack. (non-interactive mode only)")
}

// getObsSelectionFlags reads and validates the --obs-stack and --obs-service flags. Without
// --obs-service, every observability stack service is included.
func getObsSelectionFlags(cmd *cobra.Command) (obsSelection, error) {
	includeObsStack, err := cmd.Flags().GetBool("obs-stack")
	if err != nil {
		return obsSelection{}, err
	}

	services, err := cmd.Flags().GetStringArray("obs-service")
	if err != nil {
		return obsSelection{}, err
	}

	if len(services) == 0 {
		return obsSelection{includeObsStack, slices.Clone(obsServices)}, nil
	}

	if !includeObsStack {
		return obsSelection{}, errObsServiceWithoutStack
	}

	for _, service := range services {
		if !slices.Contains(obsServices, service) {
			return obsSelection{}, fmt.Errorf("%w: %q", errUnknownObsService, service)
		}
	}

	return obsSelection{includeObsStack, services}, nil
}

// resolveConnectorTarget makes sure an included admin-connector has a target, defaulting to the
// Jaeger UI when Jaeger is included.
func resolveConnectorTarget(opts *composeOptions) error {
	if !opts.includeConnector {
		return nil
	}

	if opts.connectorTarget == "" {
		if !opts.includes(jaegerName) {
			return errConnectorTargetRequired
		}

		opts.connectorTarget = defaultConnectorTarget
	}

	return validateConnectorTarget(opts.connectorTarget)
}

func validateConnectorTarget(target string) error {
	if strings.TrimSpace(target) == "" || strings.Contains(target, "/") {
		return errConnectorTargetFormat
	}

	return nil
}

func buildCompose(opts *composeOptions) string {
	var b strings.Builder

	if opts.includes(lokiName) {
		b.WriteString(lokiLoggingAnchor)
	}

	b.WriteString("services:\n")
	writePostgresService(&b, opts)
	writeSQLiteInitService(&b, opts)
	writeKerberosService(&b, opts)
	writeEchoService(&b, opts)
	writeConnectorService(&b, opts)
	writeObsServices(&b, opts)
	writeVolumes(&b, opts)

	return b.String()
}

func writePostgresService(b *strings.Builder, opts *composeOptions) {
	if !opts.includePostgres {
		return
	}

	b.WriteString(`  postgres:
    image: "postgres:18.4-alpine3.23"
    pull_policy: if_not_present
`)
	writeServiceObs(b, opts)
	b.WriteString(`    environment:
      - POSTGRES_DB=kerberos
      - POSTGRES_USER=kerberos
      - POSTGRES_PASSWORD=kerberos
    restart: on-failure
    healthcheck:
      test: ["CMD-SHELL", "psql -U kerberos -d kerberos -c 'SELECT 1' -q -t 2>/dev/null | grep -q 1"]
      interval: 1s
      timeout: 2s
      retries: 10
    volumes:
      - postgres:/var/lib/postgresql/18/docker

`)
}

func writeKerberosService(b *strings.Builder, opts *composeOptions) {
	b.WriteString(`  kerberos:
    image: "ghcr.io/trebent/kerberos:latest"
    command: --config /krb.json
    pull_policy: if_not_present
`)

	if opts.includePostgres {
		writeServiceObs(b, opts, dependency{driverPostgres, "service_healthy"})
	} else {
		writeServiceObs(b, opts, sqliteInitDependency)
	}

	b.WriteString(`    restart: on-failure
    ports:
      - 30000:30000
      - 30001:30001
    environment:
      - LOG_TO_CONSOLE=true
      - PORT=30000
      - ADMIN_PORT=30001
`)

	writeOtelEnv(b, opts, "kerberos")

	b.WriteString(`    volumes:
      - ./krb.json:/krb.json:ro
`)

	fmt.Fprintf(b, "      %s\n", krbDataMount(opts.includePostgres))
	b.WriteString("\n")
}

func writeEchoService(b *strings.Builder, opts *composeOptions) {
	if !opts.includeEcho {
		return
	}

	b.WriteString(`  echo:
    image: "ghcr.io/trebent/kerberos/echo:latest"
    pull_policy: if_not_present
`)
	writeServiceObs(b, opts)
	b.WriteString(`    restart: on-failure
    environment:
      - PORT=15000
`)

	writeOtelEnv(b, opts, "echo")
	b.WriteString("\n")
}

func writeConnectorService(b *strings.Builder, opts *composeOptions) {
	if !opts.includeConnector {
		return
	}

	b.WriteString(`  connector:
    image: "ghcr.io/trebent/kerberos/admin-connector:latest"
    command: --config /connector.json
    pull_policy: if_not_present
`)

	deps := []dependency{{defaultKRBDB, "service_started"}}
	if !opts.includePostgres {
		deps = append(deps, sqliteInitDependency)
	}

	writeServiceObs(b, opts, deps...)
	b.WriteString(`    ports:
      - 30100:30100
`)
	b.WriteString(`    restart: on-failure
    environment:
      - LOG_TO_CONSOLE=true
      - PORT=30100
`)

	fmt.Fprintf(b, "      - TARGET=%s\n", opts.connectorTarget)

	writeOtelEnv(b, opts, "connector")

	b.WriteString(`    volumes:
      - ./connector.json:/connector.json:ro
`)

	fmt.Fprintf(b, "      %s\n", krbDataMount(opts.includePostgres))
	b.WriteString("\n")
}

func writeObsServices(b *strings.Builder, opts *composeOptions) {
	writePrometheusService(b, opts)
	writeGrafanaService(b, opts)
	writeLokiService(b, opts)
	writeJaegerServices(b, opts)
}

func writePrometheusService(b *strings.Builder, opts *composeOptions) {
	if !opts.includes(prometheusName) {
		return
	}

	b.WriteString(`  prometheus:
    image: "prom/prometheus:v3"
    pull_policy: if_not_present
    command: ["--config.file=/prometheus.yml", "--storage.tsdb.path", "/prometheus/data",
              "--storage.tsdb.retention.size", "1GB"]
    restart: on-failure
`)
	writeServiceObs(b, opts)
	b.WriteString(`    volumes:
      - ./prometheus.yml:/prometheus.yml
      - prometheus:/prometheus

`)
}

func writeGrafanaService(b *strings.Builder, opts *composeOptions) {
	if !opts.includes(grafanaName) {
		return
	}

	b.WriteString(`  grafana:
    image: "grafana/grafana:13.1"
    pull_policy: if_not_present
    restart: on-failure
`)
	if opts.includePostgres {
		writeServiceObs(b, opts)
	} else {
		writeServiceObs(b, opts, sqliteInitDependency)
	}
	b.WriteString(`    ports:
      - 3000:3000
    volumes:
      - ./grafana/grafana.ini:/etc/grafana/grafana.ini
      - ./grafana/grafana-datasources.yml:/etc/grafana/provisioning/datasources/grafana-datasources.yml
      - ./grafana/grafana-dashboards.yml:/etc/grafana/provisioning/dashboards/grafana-dashboards.yml
      - ./grafana/dashboards:/var/lib/grafana/dashboards
      - grafana:/var/lib/grafana
`)
	fmt.Fprintf(b, "      %s\n\n", krbDataMount(opts.includePostgres))
}

func writeLokiService(b *strings.Builder, opts *composeOptions) {
	if !opts.includes(lokiName) {
		return
	}

	b.WriteString(`  loki:
    image: "grafana/loki:3.7"
    pull_policy: if_not_present
    command: -config.file=/etc/loki/config.yaml
    restart: on-failure
    ports:
      - 127.0.0.1:3100:3100
    volumes:
      - ./loki.yml:/etc/loki/config.yaml
      - loki:/loki

`)
}

// writeJaegerServices writes Jaeger and the job that prepares its storage volume. The Jaeger
// UI port is only published when the admin-connector does not already protect the UI.
func writeJaegerServices(b *strings.Builder, opts *composeOptions) {
	if !opts.includes(jaegerName) {
		return
	}

	b.WriteString(`  jaeger-init:
    image: busybox:1.38
    pull_policy: if_not_present
    command: ["sh", "-c", "chown 10001:0 /jaeger"]
    restart: on-failure
`)
	writeServiceObs(b, opts)
	b.WriteString(`    volumes:
      - jaeger:/jaeger

  jaeger:
    image: "jaegertracing/jaeger:2.20.0"
    pull_policy: if_not_present
`)
	writeServiceObs(b, opts, dependency{"jaeger-init", "service_completed_successfully"})
	b.WriteString(`    restart: on-failure
    command: --config /jaeger.yml
`)
	if !opts.includeConnector || opts.connectorTarget != defaultConnectorTarget {
		b.WriteString(`    ports:
      - 16686:16686
`)
	}
	b.WriteString(`    volumes:
      - ./jaeger.yml:/jaeger.yml
      - ./jaeger-config-ui.json:/jaeger-config-ui.json
      - jaeger:/jaeger

`)
}

func writeSQLiteInitService(b *strings.Builder, opts *composeOptions) {
	if opts.includePostgres {
		return
	}

	b.WriteString(`  sqlite-init:
    image: busybox:1.38
    pull_policy: if_not_present
    command: ["sh", "-c", "chmod 0777 /krbdata && touch /krbdata/krb.db && chmod 0666 /krbdata/krb.db"]
    restart: on-failure
`)
	writeServiceObs(b, opts)
	b.WriteString(`    volumes:
      - krbdata:/krbdata

`)
}

func krbDataMount(includePostgres bool) string {
	if includePostgres {
		return ""
	}

	return "- krbdata:/krbdata"
}

// writeOtelEnv writes the OpenTelemetry environment of a service: traces go to Jaeger and
// metrics are served to Prometheus when those services are included, and are off otherwise.
func writeOtelEnv(b *strings.Builder, opts *composeOptions, hostname string) {
	if opts.includes(jaegerName) {
		b.WriteString("      - OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4317\n")
		b.WriteString("      - OTEL_EXPORTER_OTLP_TRACES_PROTOCOL=grpc\n")
	}

	if opts.includes(prometheusName) {
		b.WriteString("      - OTEL_METRICS_EXPORTER=prometheus\n")
		fmt.Fprintf(b, "      - OTEL_EXPORTER_PROMETHEUS_HOST=%s\n", hostname)
		fmt.Fprintf(b, "      - OTEL_EXPORTER_PROMETHEUS_PORT=%d\n", scrapeMetricsPort)
	} else {
		b.WriteString("      - OTEL_METRICS_EXPORTER=none\n")
	}

	if !opts.includes(jaegerName) {
		b.WriteString("      - OTEL_TRACES_EXPORTER=none\n")
	}
}

func writeVolumes(b *strings.Builder, opts *composeOptions) {
	var volumes []string

	if opts.includePostgres {
		volumes = append(volumes, "  postgres:")
	} else {
		volumes = append(volumes, "  krbdata:")
	}

	for _, service := range obsServices {
		if opts.includes(service) {
			volumes = append(volumes, "  "+service+":")
		}
	}

	if len(volumes) == 0 {
		return
	}

	b.WriteString("volumes:\n")
	b.WriteString(strings.Join(volumes, "\n"))
	b.WriteString("\n")
}

// writeServiceObs writes a service's depends_on block and, when Loki is included, sends its logs
// to Loki. Loki is then added as a dependency so it starts first and stops last, giving the
// logging plugin somewhere to send each container's final log lines.
func writeServiceObs(b *strings.Builder, opts *composeOptions, deps ...dependency) {
	if opts.includes(lokiName) {
		deps = append(deps, dependency{lokiName, "service_started"})
		b.WriteString("    logging: *loki-logging\n")
	}

	if len(deps) == 0 {
		return
	}

	b.WriteString("    depends_on:\n")

	for _, d := range deps {
		fmt.Fprintf(b, "      %s:\n        condition: %s\n", d.service, d.condition)
	}
}
