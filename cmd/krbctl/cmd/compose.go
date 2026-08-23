package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// composeOptions holds the answers collected from the interactive compose session.
type composeOptions struct {
	outputPath string

	// Enablement flags.
	includeEcho      bool
	includeObsStack  bool
	includePostgres  bool
	includeConnector bool
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
	cmd.Flags().Bool("postgres", false,
		"Use PostgreSQL as the persistence backend. (non-interactive mode only)")
	cmd.Flags().Bool("connector", false,
		"Include the admin-connector service. (non-interactive mode only)")

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
	} else if err := collectComposeInteractive(opts); err != nil {
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

// collectComposeInteractive drives the interactive huh form, populating opts
// with the user's answers.
func collectComposeInteractive(opts *composeOptions) error {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Include the echo service?").
				Description("Useful for testing backends.").
				WithButtonAlignment(lipgloss.Left).
				Value(&opts.includeEcho),

			huh.NewConfirm().
				Title("Include the observability stack?").
				Description("Adds Prometheus, Grafana, and Jaeger services.").
				WithButtonAlignment(lipgloss.Left).
				Value(&opts.includeObsStack),

			huh.NewConfirm().
				Title("Include PostgreSQL as the persistence backend?").
				Description("Uses SQLite by default if skipped.").
				WithButtonAlignment(lipgloss.Left).
				Value(&opts.includePostgres),

			huh.NewConfirm().
				Title("Include the admin-connector service?").
				WithButtonAlignment(lipgloss.Left).
				Value(&opts.includeConnector),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("prompt cancelled: %w", err)
	}

	return nil
}

// collectComposeFromFlags populates opts from the command's flag values for
// non-interactive runs.
func collectComposeFromFlags(cmd *cobra.Command, opts *composeOptions) error {
	var err error

	if opts.includeEcho, err = cmd.Flags().GetBool("echo"); err != nil {
		return err
	}

	if opts.includeObsStack, err = cmd.Flags().GetBool("obs-stack"); err != nil {
		return err
	}

	if opts.includePostgres, err = cmd.Flags().GetBool("postgres"); err != nil {
		return err
	}

	if opts.includeConnector, err = cmd.Flags().GetBool("connector"); err != nil {
		return err
	}

	return nil
}

func buildCompose(opts *composeOptions) string {
	var b strings.Builder

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
    image: "ghcr.io/trebent/kerberos:latest"
    command: --config /krb.json
    pull_policy: if_not_present
`)

	if opts.includePostgres {
		b.WriteString(`    depends_on:
      postgres:
        condition: service_healthy
`)
	} else {
		b.WriteString(`    depends_on:
      sqlite-init:
        condition: service_completed_successfully
`)
	}

	b.WriteString(`    restart: on-failure
    ports:
      - 30000:30000
      - 30001:30001
    environment:
      - LOG_TO_CONSOLE=1
      - LOG_VERBOSITY=0
      - PORT=30000
      - ADMIN_PORT=30001
`)

	writeOtelEnv(b, opts.includeObsStack, "kerberos")

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
    restart: on-failure
    environment:
      - PORT=15000
`)

	writeOtelEnv(b, opts.includeObsStack, "echo")
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
    depends_on:
      kerberos:
        condition: service_started
`)
	if !opts.includePostgres {
		b.WriteString(`      sqlite-init:
        condition: service_completed_successfully
`)
	}
	b.WriteString(`    ports:
      - 30100:30100
`)
	b.WriteString(`    restart: on-failure
    environment:
      - LOG_TO_CONSOLE=true
      - LOG_VERBOSITY=0
      - PORT=30100
`)

	if opts.includeObsStack {
		fmt.Fprintf(b, "      - TARGET=jaeger:16686\n")
	}

	writeOtelEnv(b, opts.includeObsStack, "connector")

	b.WriteString(`    volumes:
      - ./connector.json:/connector.json:ro
`)

	fmt.Fprintf(b, "      %s\n", krbDataMount(opts.includePostgres))
	b.WriteString("\n")
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
    volumes:
      - ./prometheus.yml:/prometheus.yml
      - prometheus:/prometheus

  grafana:
    image: "grafana/grafana:13.1"
    pull_policy: if_not_present
    restart: on-failure
`)
	if !opts.includePostgres {
		b.WriteString(`    depends_on:
      sqlite-init:
        condition: service_completed_successfully
`)
	}
	b.WriteString(`    ports:
      - 3000:3000
    volumes:
      - ./grafana/grafana.ini:/etc/grafana/grafana.ini
      - ./grafana/grafana-datasources.yml:/etc/grafana/provisioning/datasources/grafana-datasources.yml
      - ./grafana/grafana-dashboards.yml:/etc/grafana/provisioning/dashboards/grafana-dashboards.yml
      - ./grafana/prometheus.json:/var/lib/grafana/prometheus.json
      - ./grafana/kerberos_runtime.json:/var/lib/grafana/kerberos_runtime.json
      - ./grafana/kerberos_http.json:/var/lib/grafana/kerberos_http.json
      - grafana:/var/lib/grafana
`)
	fmt.Fprintf(b, "      %s\n\n", krbDataMount(opts.includePostgres))
	b.WriteString(`  jaeger-init:
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
`)
	if !opts.includeConnector {
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
    volumes:
      - krbdata:/krbdata

`)
}

func krbDataMount(includePostgres bool) string {
	if includePostgres {
		return ""
	}

	return "- krbdata:/krbdata"
}

func writeOtelEnv(b *strings.Builder, withObs bool, hostname string) {
	if withObs {
		fmt.Fprintf(b, "      - OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4317\n")
		fmt.Fprintf(b, "      - OTEL_EXPORTER_OTLP_TRACES_PROTOCOL=grpc\n")
		fmt.Fprintf(b, "      - OTEL_METRICS_EXPORTER=prometheus\n")
		fmt.Fprintf(b, "      - OTEL_EXPORTER_PROMETHEUS_HOST=%s\n", hostname)
		fmt.Fprintf(b, "      - OTEL_EXPORTER_PROMETHEUS_PORT=9464\n")
	} else {
		b.WriteString("      - OTEL_METRICS_EXPORTER=none\n")
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
