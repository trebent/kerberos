package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

const (
	driverPostgres         = "postgres"
	driverSQLite           = "sqlite"
	defaultKRBDB           = "kerberos"
	defaultConnectorTarget = "http://kerberos:30001"
)

// configOptions holds all answers collected from the interactive config session.
type configOptions struct {
	// Kerberos gateway
	backends        []backendEntry
	includeAuth     bool
	persistenceMode string // "sqlite" or "postgres"
	outputPath      string

	// Observability
	includeObs bool
	obsOpts    obsConfigOptions

	// Admin-connector
	includeConnector bool
	connectorOpts    connectorOptions
}

type backendEntry struct {
	name string
	host string
	port int
}

// obsConfigOptions holds the answers for the observability config section.
type obsConfigOptions struct {
	scrapeTargets    []string // e.g. ["kerberos","echo","connector","jaeger"]
	grafanaDB        string   // "postgres" or "sqlite"
	grafanaAnonymous bool
}

// connectorOptions holds the answers for the admin-connector config section.
type connectorOptions struct {
	targetURL       string
	corsOrigin      string
	persistenceMode string // "sqlite" or "postgres"
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Interactively generate a base Kerberos configuration file",
		Long: `Walks you through a series of prompts to build a base Kerberos JSON
configuration file. Mandatory sections are always included; optional sections
(auth, observability, postgres persistence, admin-connector) can be skipped.`,
		RunE: runConfig,
	}

	cmd.Flags().StringP("output", "o", "krb.json", "Path to write the generated krb.json")

	return cmd
}

func runConfig(cmd *cobra.Command, _ []string) error {
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}

	opts := &configOptions{
		outputPath: output,
		connectorOpts: connectorOptions{
			targetURL:       defaultConnectorTarget,
			corsOrigin:      defaultConnectorTarget,
			persistenceMode: driverSQLite,
		},
		obsOpts: obsConfigOptions{
			grafanaDB:        driverPostgres,
			grafanaAnonymous: true,
		},
	}

	if err := promptKerberosSection(opts); err != nil {
		return err
	}

	if opts.includeObs {
		if err := promptObsSection(opts); err != nil {
			return err
		}
	}

	if opts.includeConnector {
		if err := promptConnectorSection(opts); err != nil {
			return err
		}
	}

	// Write krb.json
	content, err := buildConfig(opts)
	if err != nil {
		return fmt.Errorf("failed to build config: %w", err)
	}

	if err := os.WriteFile(opts.outputPath, content, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Fprintf(os.Stdout, "krb.json written to %s\n", opts.outputPath)

	// Write observability config files
	if opts.includeObs {
		if err := writeObsFiles(opts); err != nil {
			return err
		}
	}

	// Write connector.json
	if opts.includeConnector {
		connContent, err := buildConnectorJSON(&opts.connectorOpts)
		if err != nil {
			return fmt.Errorf("failed to build connector config: %w", err)
		}

		if err := os.WriteFile("connector.json", connContent, 0o600); err != nil {
			return fmt.Errorf("failed to write connector.json: %w", err)
		}

		fmt.Fprintln(os.Stdout, "connector.json written to connector.json")
	}

	return nil
}

// promptKerberosSection runs the main Kerberos gateway configuration prompts.
func promptKerberosSection(opts *configOptions) error {
	if err := promptBackendsHuh(opts); err != nil {
		return err
	}

	persistenceOpts := []huh.Option[string]{
		huh.NewOption("SQLite (default, file-based)", driverSQLite),
		huh.NewOption("PostgreSQL", driverPostgres),
	}
	opts.persistenceMode = driverSQLite

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Include the auth section?").
				Description("Enables basic authentication for backend routes.").
				Value(&opts.includeAuth),

			huh.NewConfirm().
				Title("Include the observability section?").
				Description("Enables metrics and tracing for Kerberos.").
				Value(&opts.includeObs),

			huh.NewSelect[string]().
				Title("Persistence backend").
				Options(persistenceOpts...).
				Value(&opts.persistenceMode),

			huh.NewConfirm().
				Title("Include the admin-connector?").
				Description("Generates connector.json for the admin-connector service.").
				Value(&opts.includeConnector),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("prompt cancelled: %w", err)
	}

	return nil
}

// promptBackendsHuh collects one or more backend target entries using huh.
func promptBackendsHuh(opts *configOptions) error {
	for {
		var (
			name    string
			host    string
			portStr string
		)

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title(fmt.Sprintf("Backend %d — name", len(opts.backends)+1)).
					Description(`Press Enter with an empty name to finish (at least one required).`).
					Placeholder("my-api").
					Value(&name),
			),
		)

		if err := form.Run(); err != nil {
			return fmt.Errorf("prompt cancelled: %w", err)
		}

		name = strings.TrimSpace(name)
		if name == "" {
			if len(opts.backends) == 0 {
				fmt.Fprintln(os.Stderr, "At least one backend is required.")
				continue
			}

			break
		}

		detailForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title(fmt.Sprintf("Backend %q — host", name)).
					Placeholder("localhost").
					Value(&host),

				huh.NewInput().
					Title(fmt.Sprintf("Backend %q — port", name)).
					Placeholder("8080").
					Value(&portStr),
			),
		)

		if err := detailForm.Run(); err != nil {
			return fmt.Errorf("prompt cancelled: %w", err)
		}

		if strings.TrimSpace(host) == "" {
			host = "localhost"
		}

		port := parsePort(portStr)
		opts.backends = append(opts.backends, backendEntry{name: name, host: host, port: port})

		var addAnother bool
		confirmForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Add another backend?").
					Value(&addAnother),
			),
		)

		if err := confirmForm.Run(); err != nil {
			return fmt.Errorf("prompt cancelled: %w", err)
		}

		if !addAnother {
			break
		}
	}

	return nil
}

// promptObsSection runs the observability configuration prompts.
func promptObsSection(opts *configOptions) error {
	opts.obsOpts.scrapeTargets = []string{defaultKRBDB}

	grafanaDBOpts := []huh.Option[string]{
		huh.NewOption("PostgreSQL", driverPostgres),
		huh.NewOption("SQLite (Grafana default)", "sqlite3"),
	}

	scrapeOpts := []huh.Option[string]{
		huh.NewOption("kerberos (port 9464)", "kerberos"),
		huh.NewOption("echo (port 9463)", "echo"),
		huh.NewOption("connector (port 9462)", "connector"),
		huh.NewOption("jaeger (port 8888)", "jaeger"),
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Prometheus scrape targets").
				Description("Select services that should expose metrics to Prometheus.").
				Options(scrapeOpts...).
				Value(&opts.obsOpts.scrapeTargets),

			huh.NewSelect[string]().
				Title("Grafana database backend").
				Options(grafanaDBOpts...).
				Value(&opts.obsOpts.grafanaDB),

			huh.NewConfirm().
				Title("Enable Grafana anonymous access?").
				Description("Allows viewing dashboards without logging in.").
				Value(&opts.obsOpts.grafanaAnonymous),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("prompt cancelled: %w", err)
	}

	return nil
}

// promptConnectorSection runs the admin-connector configuration prompts.
func promptConnectorSection(opts *configOptions) error {
	persistenceOpts := []huh.Option[string]{
		huh.NewOption("SQLite (default, file-based)", driverSQLite),
		huh.NewOption("PostgreSQL", driverPostgres),
	}
	opts.connectorOpts.persistenceMode = driverSQLite

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Kerberos admin target URL").
				Description("The URL at which the admin-connector can reach Kerberos.").
				Placeholder(defaultConnectorTarget).
				Value(&opts.connectorOpts.targetURL),

			huh.NewInput().
				Title("Allowed CORS origin").
				Description("The origin browsers are served from (used to allow cross-origin requests).").
				Placeholder(defaultConnectorTarget).
				Value(&opts.connectorOpts.corsOrigin),

			huh.NewSelect[string]().
				Title("Connector persistence backend").
				Options(persistenceOpts...).
				Value(&opts.connectorOpts.persistenceMode),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("prompt cancelled: %w", err)
	}

	if strings.TrimSpace(opts.connectorOpts.targetURL) == "" {
		opts.connectorOpts.targetURL = defaultConnectorTarget
	}

	if strings.TrimSpace(opts.connectorOpts.corsOrigin) == "" {
		opts.connectorOpts.corsOrigin = defaultConnectorTarget
	}

	return nil
}

func parsePort(raw string) int {
	const defaultPort = 8080

	port, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || port < 1 || port > 65535 {
		return defaultPort
	}

	return port
}

//nolint:cyclop // config builder requires branching per optional section
func buildConfig(opts *configOptions) ([]byte, error) {
	backends := make([]map[string]any, 0, len(opts.backends))
	for _, b := range opts.backends {
		backends = append(backends, map[string]any{
			"name": b.name,
			"host": b.host,
			"port": b.port,
		})
	}

	root := map[string]any{
		"gateway": map[string]any{
			"router": map[string]any{
				"backends": backends,
			},
		},
	}

	if opts.includeObs {
		root["observability"] = map[string]any{
			"enabled":        true,
			"runtimeMetrics": true,
		}
	}

	if opts.includeAuth && len(opts.backends) > 0 {
		root["auth"] = buildAuthSection(opts.backends)
	}

	root["persistence"] = buildPersistenceSection(opts.persistenceMode)

	return json.MarshalIndent(root, "", "  ")
}

func buildAuthSection(backends []backendEntry) map[string]any {
	mappings := make([]map[string]any, 0, len(backends))
	for _, b := range backends {
		mappings = append(mappings, map[string]any{
			"backend": b.name,
			"method":  "basic",
			"exempt":  []string{},
		})
	}

	return map[string]any{
		"methods": map[string]any{
			"basic": map[string]any{},
		},
		"scheme": map[string]any{
			"mappings": mappings,
		},
		"order": 1,
	}
}

func buildPersistenceSection(mode string) map[string]any {
	if mode == driverPostgres {
		return map[string]any{
			"driver":  driverPostgres,
			"address": "postgres:5432",
			"postgres": map[string]any{
				"database": defaultKRBDB,
				"username": defaultKRBDB,
				"password": defaultKRBDB,
				"sslMode":  "disable",
			},
		}
	}

	return map[string]any{
		"driver":  driverSQLite,
		"address": "krb.db",
	}
}
