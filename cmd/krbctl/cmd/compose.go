package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// composeOptions holds the answers collected from the interactive compose session.
type composeOptions struct {
	includeEcho      bool
	includeObsStack  bool
	includePostgres  bool
	includeConnector bool
	outputPath       string
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

	cmd.Flags().StringP("output", "o", "compose.yaml", "Path to write the generated compose.yaml")

	return cmd
}

func runCompose(cmd *cobra.Command, _ []string) error {
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(os.Stdin)
	opts := &composeOptions{outputPath: output}

	fmt.Fprintln(os.Stdout, "=== Kerberos compose.yaml generator ===")
	fmt.Fprintln(os.Stdout)

	opts.includeEcho = promptYesNo(scanner,
		"Include the echo service (useful for testing backends)? [y/N]")

	opts.includeObsStack = promptYesNo(scanner,
		"Include the observability stack (Prometheus, Grafana, Jaeger)? [y/N]")

	opts.includePostgres = promptYesNo(scanner,
		"Include PostgreSQL as the persistence backend? [y/N]")

	opts.includeConnector = promptYesNo(scanner,
		"Include the admin-connector service? [y/N]")

	content := buildCompose(opts)

	if err := os.WriteFile(opts.outputPath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("failed to write compose file: %w", err)
	}

	fmt.Fprintf(os.Stdout, "\ncompose.yaml written to %s\n", opts.outputPath)

	return nil
}

// promptYesNo prints the question and reads a y/n answer from the scanner.
// Returns true for "y"/"yes", false otherwise (default: false).
func promptYesNo(scanner *bufio.Scanner, question string) bool {
	fmt.Fprint(os.Stdout, question+" ")

	if scanner.Scan() {
		switch strings.TrimSpace(strings.ToLower(scanner.Text())) {
		case "y", "yes":
			return true
		default:
			return false
		}
	}

	return false
}

func buildCompose(opts *composeOptions) string {
	var b strings.Builder

	b.WriteString("services:\n")
	writePostgresService(&b, opts)
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
    environment:
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
    image: "ghcr.io/trebent/kerberos:${VERSION:-unset}"
    command: --config /krb.json
    pull_policy: if_not_present
`)

	if opts.includePostgres {
		b.WriteString(`    depends_on:
      postgres:
        condition: service_healthy
`)
	}

	b.WriteString(`    restart: on-failure
    ports:
      - ${KERBEROS_PORT:-30000}:${KERBEROS_PORT:-30000}
      - ${KERBEROS_ADMIN_PORT:-30001}:${KERBEROS_ADMIN_PORT:-30001}
`)

	if opts.includeObsStack {
		b.WriteString("      - ${KERBEROS_METRICS_PORT:-9464}:${KERBEROS_METRICS_PORT:-9464}\n")
	}

	b.WriteString(`    environment:
      - LOG_TO_CONSOLE=1
      - LOG_VERBOSITY=${LOG_VERBOSITY:-20}
      - PORT=${KERBEROS_PORT:-30000}
      - ADMIN_PORT=${KERBEROS_ADMIN_PORT:-30001}
      - VERSION=${VERSION:-unset}
`)

	writeOtelEnv(b, opts.includeObsStack, "kerberos", "${KERBEROS_METRICS_PORT:-9464}")

	b.WriteString(`    volumes:
      - ./krb.json:/krb.json:ro

`)
}

func writeEchoService(b *strings.Builder, opts *composeOptions) {
	if !opts.includeEcho {
		return
	}

	b.WriteString(`  echo:
    image: "ghcr.io/trebent/kerberos/echo:${VERSION:-unset}"
    pull_policy: if_not_present
    restart: on-failure
    ports:
      - ${ECHO_PORT:-15000}:${ECHO_PORT:-15000}
`)

	if opts.includeObsStack {
		b.WriteString("      - ${ECHO_METRICS_PORT:-9463}:${ECHO_METRICS_PORT:-9463}\n")
	}

	b.WriteString(`    environment:
      - PORT=${ECHO_PORT:-15000}
`)

	writeOtelEnv(b, opts.includeObsStack, "echo", "${ECHO_METRICS_PORT:-9463}")
	b.WriteString("\n")
}

func writeConnectorService(b *strings.Builder, opts *composeOptions) {
	if !opts.includeConnector {
		return
	}

	b.WriteString(`  connector:
    image: "ghcr.io/trebent/kerberos/admin-connector:${VERSION:-unset}"
    command: --config /connector.json
    pull_policy: if_not_present
    depends_on:
      kerberos:
        condition: service_started
    restart: on-failure
    ports:
      - ${CONNECTOR_PORT:-30100}:${CONNECTOR_PORT:-30100}
`)

	if opts.includeObsStack {
		b.WriteString("      - ${CONNECTOR_METRICS_PORT:-9462}:${CONNECTOR_METRICS_PORT:-9462}\n")
	}

	b.WriteString(`    environment:
      - LOG_TO_CONSOLE=true
      - LOG_VERBOSITY=${LOG_VERBOSITY:-20}
      - VERSION=${VERSION:-unset}
      - PORT=${CONNECTOR_PORT:-30100}
`)

	writeOtelEnv(b, opts.includeObsStack, "connector", "${CONNECTOR_METRICS_PORT:-9462}")

	b.WriteString(`    volumes:
      - ./connector.json:/connector.json:ro

`)
}

func writeObsServices(b *strings.Builder, opts *composeOptions) {
	if !opts.includeObsStack {
		return
	}

	b.WriteString(`  prometheus:
    image: "prom/prometheus:v3"
    pull_policy: if_not_present
    command: ["--config.file=/prometheus.yml", "--storage.tsdb.path", "/prometheus/data",
              "--storage.tsdb.retention.size", "1GB"]
    restart: on-failure
    ports:
      - ${PROM_PORT:-9090}:9090
    volumes:
      - ./prometheus.yml:/prometheus.yml
      - prometheus:/prometheus

  grafana:
    image: "grafana/grafana:13.1"
    pull_policy: if_not_present
    restart: on-failure
    ports:
      - ${GRAFANA_PORT:-3000}:3000
    volumes:
      - ./grafana/grafana-datasources.yml:/etc/grafana/provisioning/datasources/grafana-datasources.yml
      - ./grafana/grafana-dashboards.yml:/etc/grafana/provisioning/dashboards/grafana-dashboards.yml
      - grafana:/var/lib/grafana

  jaeger-init:
    image: busybox:1.38
    pull_policy: if_not_present
    command: ["sh", "-c", "chown 10001:0 /jaeger"]
    restart: on-failure
    volumes:
      - jaeger:/jaeger

  jaeger:
    image: "jaegertracing/jaeger:2.20.0"
    pull_policy: if_not_present
    depends_on:
      jaeger-init:
        condition: service_completed_successfully
    restart: on-failure
    command: --config /jaeger.yml
    ports:
      - 16686:16686
    volumes:
      - ./jaeger.yml:/jaeger.yml
      - jaeger:/jaeger

`)
}

func writeOtelEnv(b *strings.Builder, withObs bool, hostname, metricsPort string) {
	if withObs {
		fmt.Fprintf(b, "      - OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4317\n")
		fmt.Fprintf(b, "      - OTEL_EXPORTER_OTLP_TRACES_PROTOCOL=grpc\n")
		fmt.Fprintf(b, "      - OTEL_METRICS_EXPORTER=prometheus\n")
		fmt.Fprintf(b, "      - OTEL_EXPORTER_PROMETHEUS_HOST=%s\n", hostname)
		fmt.Fprintf(b, "      - OTEL_EXPORTER_PROMETHEUS_PORT=%s\n", metricsPort)
	} else {
		b.WriteString("      - OTEL_METRICS_EXPORTER=none\n")
		b.WriteString("      - OTEL_TRACES_EXPORTER=none\n")
	}
}

func writeVolumes(b *strings.Builder, opts *composeOptions) {
	var volumes []string

	if opts.includePostgres {
		volumes = append(volumes, "  postgres:")
	}

	if opts.includeObsStack {
		volumes = append(volumes, "  prometheus:", "  grafana:", "  jaeger:")
	}

	if len(volumes) == 0 {
		return
	}

	b.WriteString("volumes:\n")
	b.WriteString(strings.Join(volumes, "\n"))
	b.WriteString("\n")
}
