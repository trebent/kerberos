package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

const (
	driverPostgres  = "postgres"
	driverSQLite    = "sqlite"
	defaultKRBDB    = "kerberos"
	echoBackendName = "echo"
	echoBackendHost = "echo"
	echoBackendPort = 15000
	jaegerName      = "jaeger"

	// scrapeMetricsPort is the default Prometheus exporter port every service
	// exposes its metrics on.
	scrapeMetricsPort = 9464
	// sqliteSharedPath is the SQLite file location on the shared krbdata volume,
	// letting Kerberos and the admin-connector share one database file.
	sqliteSharedPath = "/krbdata/krb.db"
)

// configOptions holds all answers collected from the interactive config session.
type configOptions struct {
	outputPath string

	// Kerberos gateway
	backends []backendEntry

	// persistence driver selection
	driver string // "sqlite" or "postgres"

	// Observability stack (Prometheus/Grafana/Jaeger) config generation
	includeObsStack bool
	obsOpts         obsConfigOptions

	// Admin-connector
	includeConnector bool
	connectorOpts    connectorOptions
}

type backendEntry struct {
	name string
	host string
	port int
	auth bool
}

// obsConfigOptions holds the answers for the observability stack config section.
type obsConfigOptions struct {
	scrapeTargets    []string // e.g. ["kerberos","echo","connector","jaeger"]
	grafanaAnonymous bool
}

// connectorOptions holds the answers for the admin-connector config section.
type connectorOptions struct {
	allowAllOrigins bool
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Interactively generate a base Kerberos configuration file",
		Long: `Walks you through a series of prompts to build a base Kerberos JSON
configuration file. Mandatory sections are always included (Kerberos
observability is always enabled); optional sections (per-backend auth,
observability-stack config, postgres persistence, admin-connector) can be
skipped.`,
		RunE: runConfig,
	}

	cmd.Flags().StringP("output", "o", ".", "Output path where config files will be written.")

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
			allowAllOrigins: true,
		},
		driver: driverSQLite,
		obsOpts: obsConfigOptions{
			grafanaAnonymous: true,
		},
	}

	if err := promptBackends(opts); err != nil {
		return err
	}

	if err := promptFixedSections(opts); err != nil {
		return err
	}

	// Write krb.json
	content, err := buildConfig(opts)
	if err != nil {
		return fmt.Errorf("failed to build config: %w", err)
	}

	if err := os.WriteFile(
		filepath.Join(opts.outputPath, "krb.json"), content, 0o644,
	); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Fprintf(os.Stdout, "krb.json written to %s\n", opts.outputPath)

	// Write observability stack config files
	if opts.includeObsStack {
		if err := writeObsFiles(opts.driver, opts); err != nil {
			return err
		}
	}

	// Write connector.json
	if opts.includeConnector {
		connContent, err := buildConnectorJSON(opts.driver, &opts.connectorOpts)
		if err != nil {
			return fmt.Errorf("failed to build connector config: %w", err)
		}

		if err := os.WriteFile(
			filepath.Join(opts.outputPath, "connector.json"), connContent, 0o644,
		); err != nil {
			return fmt.Errorf("failed to write connector.json: %w", err)
		}

		fmt.Fprintln(os.Stdout, "connector.json written to connector.json")
	}

	return nil
}

// promptBackends collects one or more backend target entries using huh. The echo
// service can be registered as a router backend up front; when present the user
// may finish without registering any manual backend.
func promptBackends(opts *configOptions) error {
	if err := promptEchoBackend(opts); err != nil {
		return err
	}

	for {
		name, err := promptBackendName(opts)
		if err != nil {
			return err
		}

		if name == "" {
			break
		}

		if err := promptBackendDetails(opts, name); err != nil {
			return err
		}

		addAnother, err := promptAddAnother()
		if err != nil {
			return err
		}

		if !addAnother {
			break
		}
	}

	return nil
}

// promptEchoBackend asks whether to register the echo service as a backend.
func promptEchoBackend(opts *configOptions) error {
	var useEcho bool

	echoForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Use the echo service as a backend?").
				Description(fmt.Sprintf(
					"Registers echo (%s:%d) as a router backend.",
					echoBackendHost, echoBackendPort,
				)).
				WithButtonAlignment(lipgloss.Left).
				Value(&useEcho),
		),
	)

	if err := echoForm.Run(); err != nil {
		return fmt.Errorf("prompt cancelled: %w", err)
	}

	if useEcho {
		opts.backends = append(opts.backends, backendEntry{
			name: echoBackendName,
			host: echoBackendHost,
			port: echoBackendPort,
		})
	}

	return nil
}

// promptBackendName asks for a backend name. An empty result signals the user is
// done. At least one backend is required, enforced via inline validation.
func promptBackendName(opts *configOptions) (string, error) {
	var name string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(fmt.Sprintf("Backend %d — name", len(opts.backends)+1)).
				Description(`Press Enter with an empty name to finish (at least one required).`).
				Placeholder("my-api").
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" && len(opts.backends) == 0 {
						return errors.New("at least one backend is required")
					}

					return nil
				}).
				Value(&name),
		),
	)

	if err := form.Run(); err != nil {
		return "", fmt.Errorf("prompt cancelled: %w", err)
	}

	return strings.TrimSpace(name), nil
}

// promptBackendDetails asks for the host and port of the named backend and
// appends it to the backend list.
func promptBackendDetails(opts *configOptions, name string) error {
	var (
		host    string
		portStr string
	)

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

	opts.backends = append(opts.backends, backendEntry{
		name: name,
		host: host,
		port: parsePort(portStr),
	})

	return nil
}

// promptAddAnother asks whether to register another backend.
func promptAddAnother() (bool, error) {
	var addAnother bool

	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Add another backend?").
				WithButtonAlignment(lipgloss.Left).
				Value(&addAnother),
		),
	)

	if err := confirmForm.Run(); err != nil {
		return false, fmt.Errorf("prompt cancelled: %w", err)
	}

	return addAnother, nil
}

// promptFixedSections runs the remaining configuration prompts as a single
// multi-group form so the user can move back and forth between sections with
// shift+tab. Conditional groups are hidden until their toggle is enabled.
func promptFixedSections(opts *configOptions) error {
	opts.obsOpts.scrapeTargets = defaultScrapeTargets(opts)

	form := huh.NewForm(
		buildPersistenceGroup(opts),
		buildAuthGroup(opts),
		buildObsToggleGroup(opts),
		buildObsOptionsGroup(opts),
		buildConnectorToggleGroup(opts),
		buildConnectorConfigGroup(opts),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("prompt cancelled: %w", err)
	}

	return nil
}

// defaultScrapeTargets returns the scrape targets pre-selected by default:
// kerberos, jaeger, and every registered router backend.
func defaultScrapeTargets(opts *configOptions) []string {
	targets := []string{defaultKRBDB, jaegerName}
	for _, b := range opts.backends {
		targets = append(targets, b.name)
	}

	return targets
}

// buildAuthGroup builds a dedicated authentication section with a per-backend
// toggle enabling basic auth for that backend.
func buildAuthGroup(opts *configOptions) *huh.Group {
	fields := make([]huh.Field, 0, len(opts.backends))
	for i := range opts.backends {
		fields = append(fields, huh.NewConfirm().
			Title(fmt.Sprintf("Enable basic auth for backend %q?", opts.backends[i].name)).
			WithButtonAlignment(lipgloss.Left).
			Value(&opts.backends[i].auth))
	}

	return huh.NewGroup(fields...).Title("Authentication")
}

// buildObsToggleGroup builds the dedicated toggle controlling generation of the
// observability stack (Prometheus/Grafana/Jaeger) config files.
func buildObsToggleGroup(opts *configOptions) *huh.Group {
	return huh.NewGroup(
		huh.NewConfirm().
			Title("Generate observability stack config?").
			Description("Writes Prometheus, Grafana, and Jaeger config files for the " +
				"observability stack. (Kerberos observability itself is always enabled.)").
			WithButtonAlignment(lipgloss.Left).
			Value(&opts.includeObsStack),
	).Title("Observability stack")
}

// buildObsOptionsGroup builds the observability stack detail options, hidden
// unless the observability stack toggle is enabled.
func buildObsOptionsGroup(opts *configOptions) *huh.Group {
	scrapeOpts := []huh.Option[string]{
		huh.NewOption(fmt.Sprintf("kerberos (port %d)", scrapeMetricsPort), defaultKRBDB),
		huh.NewOption("jaeger (port 8888)", jaegerName),
	}
	for _, b := range opts.backends {
		scrapeOpts = append(scrapeOpts, huh.NewOption(
			fmt.Sprintf("%s (%s:%d)", b.name, b.host, scrapeMetricsPort), b.name))
	}

	return huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title("Prometheus scrape targets").
			Description("Select services that should expose metrics to Prometheus. "+
				"The admin-connector is scraped automatically when included.").
			Options(scrapeOpts...).
			Value(&opts.obsOpts.scrapeTargets),

		huh.NewConfirm().
			Title("Enable Grafana anonymous access?").
			Description("Allows viewing dashboards without logging in.").
			WithButtonAlignment(lipgloss.Left).
			Value(&opts.obsOpts.grafanaAnonymous),
	).Title("Observability stack options").
		WithHideFunc(func() bool { return !opts.includeObsStack })
}

// buildPersistenceGroup builds the Kerberos persistence selection section.
func buildPersistenceGroup(opts *configOptions) *huh.Group {
	persistenceOpts := []huh.Option[string]{
		huh.NewOption("SQLite (default, file-based)", driverSQLite),
		huh.NewOption("PostgreSQL", driverPostgres),
	}

	return huh.NewGroup(
		huh.NewSelect[string]().
			Title("Persistence backend").
			Options(persistenceOpts...).
			Value(&opts.driver),
	).Title("Persistence")
}

// buildConnectorToggleGroup builds the dedicated admin-connector on/off section.
func buildConnectorToggleGroup(opts *configOptions) *huh.Group {
	return huh.NewGroup(
		huh.NewConfirm().
			Title("Include the admin-connector?").
			Description("Generates connector.json for the admin-connector service.").
			WithButtonAlignment(lipgloss.Left).
			Value(&opts.includeConnector),
	).Title("Admin-connector")
}

// buildConnectorConfigGroup builds the admin-connector config detail section,
// hidden unless the admin-connector toggle is enabled.
func buildConnectorConfigGroup(opts *configOptions) *huh.Group {
	return huh.NewGroup(
		huh.NewConfirm().
			Title("Allow all CORS origins?").
			Description("Yes allows any origin; No denies all cross-origin requests.").
			WithButtonAlignment(lipgloss.Left).
			Value(&opts.connectorOpts.allowAllOrigins),
	).Title("Admin-connector config").
		WithHideFunc(func() bool { return !opts.includeConnector })
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

	root["observability"] = map[string]any{
		"enabled":        true,
		"runtimeMetrics": true,
	}

	if authBackends := authEnabledBackends(opts.backends); len(authBackends) > 0 {
		root["auth"] = buildAuthSection(authBackends)
	}

	root["persistence"] = buildPersistenceSection(opts.driver)

	return json.MarshalIndent(root, "", "  ")
}

// authEnabledBackends returns the subset of backends that have auth enabled.
func authEnabledBackends(backends []backendEntry) []backendEntry {
	enabled := make([]backendEntry, 0, len(backends))
	for _, b := range backends {
		if b.auth {
			enabled = append(enabled, b)
		}
	}

	return enabled
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
		"address": sqliteSharedPath,
	}
}
