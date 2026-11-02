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
	lokiName        = "loki"
	connectorName   = "connector"

	// defaultBackendHost and defaultBackendPort are used when a backend is
	// registered without an explicit host or port.
	defaultBackendHost = "localhost"
	defaultBackendPort = 8080

	// scrapeMetricsPort is the default Prometheus exporter port every service
	// exposes its metrics on.
	scrapeMetricsPort = 9464
	// sqliteSharedPath is the SQLite file location on the shared krbdata volume,
	// letting Kerberos and the admin-connector share one database file.
	sqliteSharedPath = "/krbdata/krb.db"
)

var (
	errDuplicateBackend = errors.New("duplicate backend name")
	errInvalidPort      = errors.New("port must be a number between 1 and 65535")
)

// addDriverFlag registers the --driver flag shared by the config and compose
// commands.
func addDriverFlag(cmd *cobra.Command) {
	cmd.Flags().String("driver", driverSQLite,
		"Persistence driver: sqlite or postgres. (non-interactive mode only)")
}

// getDriverFlag reads and validates the --driver flag.
func getDriverFlag(cmd *cobra.Command) (string, error) {
	driver, err := cmd.Flags().GetString("driver")
	if err != nil {
		return "", err
	}

	if driver != driverSQLite && driver != driverPostgres {
		return "", fmt.Errorf("invalid driver %q: must be %q or %q",
			driver, driverSQLite, driverPostgres)
	}

	return driver, nil
}

// configOptions holds all answers collected from the interactive config session.
type configOptions struct {
	outputPath string

	// Kerberos gateway
	backends []backendEntry

	// persistence driver selection
	driver string // "sqlite" or "postgres"

	// Observability stack (Prometheus/Grafana/Jaeger/Loki) config generation
	obsSelection
	obsOpts obsConfigOptions

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
configuration file. You first choose what to include (echo backend,
persistence, basic auth, admin-connector, observability stack), then you are
asked for the details of each part you chose.`,
		RunE: runConfig,
	}

	cmd.Flags().StringP("output", "o", ".", "Output path where config files will be written.")
	cmd.Flags().BoolP("non-interactive", "y", false,
		"Skip prompts and build the configuration from flag values.")
	cmd.Flags().Bool("echo", false,
		"Register the built-in echo service as a router backend. (non-interactive mode only)")
	cmd.Flags().StringArray("backend", nil,
		"Router backend as 'name=..,host=..,port=..,auth=..' (repeatable). (non-interactive mode only)")
	addDriverFlag(cmd)
	cmd.Flags().Bool("obs-stack", false,
		"Generate observability stack config files. (non-interactive mode only)")
	addObsServiceFlag(cmd)
	cmd.Flags().Bool("grafana-anonymous", true,
		"Enable Grafana anonymous access. (non-interactive mode only)")
	cmd.Flags().StringArray("scrape-target", nil,
		"Prometheus scrape target (repeatable); defaults to kerberos, all backends, and jaeger, "+
			"loki and the connector when included. "+
			"(non-interactive mode only)")
	cmd.Flags().Bool("connector", false,
		"Include the admin-connector (generates connector.json). (non-interactive mode only)")
	cmd.Flags().Bool("connector-allow-all-origins", true,
		"Allow all CORS origins for the admin-connector. (non-interactive mode only)")

	return cmd
}

func runConfig(cmd *cobra.Command, _ []string) error {
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}

	nonInteractive, err := cmd.Flags().GetBool("non-interactive")
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

	if nonInteractive {
		if err := collectConfigFromFlags(cmd, opts); err != nil {
			return err
		}
	} else if err := runInteractive(cmd, func() error {
		return collectConfigInteractive(opts)
	}); err != nil {
		return err
	}

	// Write krb.json
	content, err := buildConfig(opts)
	if err != nil {
		return fmt.Errorf("failed to build config: %w", err)
	}

	//nolint:gosec // welp
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

		//nolint:gosec // welp
		if err := os.WriteFile(
			filepath.Join(opts.outputPath, "connector.json"), connContent, 0o644,
		); err != nil {
			return fmt.Errorf("failed to write connector.json: %w", err)
		}

		fmt.Fprintf(os.Stdout, "connector.json written to %s\n", opts.outputPath)
	}

	return nil
}

// optInChoices holds the answers from the opt-in form that steer the later prompts but are
// not stored in configOptions directly.
type optInChoices struct {
	useEcho     bool
	usePostgres bool
	basicAuth   bool
}

// collectConfigInteractive drives the interactive prompt sequence, populating
// opts with the user's answers. A single opt-in form asks what to include, then
// the prompts dig down into each section that was opted in.
func collectConfigInteractive(opts *configOptions) error {
	var choices optInChoices

	if err := runForm(buildOptInGroup(opts, &choices)); err != nil {
		return err
	}

	if err := promptObsServices(&opts.obsSelection); err != nil {
		return err
	}

	if choices.useEcho {
		opts.backends = append(opts.backends, echoBackend())
	}

	if choices.usePostgres {
		opts.driver = driverPostgres
	}

	if err := promptBackends(opts); err != nil {
		return err
	}

	if choices.basicAuth {
		for i := range opts.backends {
			opts.backends[i].auth = true
		}
	}

	return promptSectionSettings(opts)
}

// buildOptInGroup builds the first form, where the user chooses which parts of the
// deployment to include.
func buildOptInGroup(opts *configOptions, choices *optInChoices) *huh.Group {
	return huh.NewGroup(
		newConfirm(
			"Use the echo service as a backend?",
			fmt.Sprintf(
				"Registers echo (%s:%d) as a router backend.", echoBackendHost, echoBackendPort,
			),
			&choices.useEcho,
		),
		newPostgresConfirm(&choices.usePostgres),
		newConfirm(
			"Enable basic auth?",
			"Puts every backend behind basic authentication.",
			&choices.basicAuth,
		),
		newConfirm(
			"Include the admin-connector?",
			"Generates connector.json for the admin-connector service.",
			&opts.includeConnector,
		),
		newConfirm(
			"Generate observability stack config?",
			"Choose from Prometheus, Grafana, Jaeger, and Loki next.",
			&opts.includeObsStack,
		),
	)
}

// promptSectionSettings asks for the settings of every opted-in section in one form, so the
// user can move between sections with shift+tab.
func promptSectionSettings(opts *configOptions) error {
	groups := make([]*huh.Group, 0, 2)

	if opts.includeConnector {
		groups = append(groups, buildConnectorConfigGroup(opts))
	}

	if opts.includeObsStack {
		opts.obsOpts.scrapeTargets = defaultScrapeTargets(opts)

		if group := buildObsOptionsGroup(opts); group != nil {
			groups = append(groups, group)
		}
	}

	if len(groups) == 0 {
		return nil
	}

	return runForm(groups...)
}

// collectConfigFromFlags populates opts from the command's flag values for
// non-interactive runs.
func collectConfigFromFlags(cmd *cobra.Command, opts *configOptions) error {
	useEcho, err := cmd.Flags().GetBool("echo")
	if err != nil {
		return err
	}

	if useEcho {
		opts.backends = append(opts.backends, echoBackend())
	}

	backendSpecs, err := cmd.Flags().GetStringArray("backend")
	if err != nil {
		return err
	}

	for _, spec := range backendSpecs {
		backend, err := parseBackendFlag(spec)
		if err != nil {
			return fmt.Errorf("parse backend %q: %w", spec, err)
		}

		if hasBackend(opts.backends, backend.name) {
			return fmt.Errorf("%w: %q", errDuplicateBackend, backend.name)
		}

		opts.backends = append(opts.backends, backend)
	}

	if len(opts.backends) == 0 {
		return errors.New("at least one backend is required (use --echo or --backend)")
	}

	if opts.driver, err = getDriverFlag(cmd); err != nil {
		return err
	}

	if opts.obsSelection, err = getObsSelectionFlags(cmd); err != nil {
		return err
	}

	if opts.obsOpts.grafanaAnonymous, err = cmd.Flags().GetBool("grafana-anonymous"); err != nil {
		return err
	}

	if opts.includeConnector, err = cmd.Flags().GetBool("connector"); err != nil {
		return err
	}

	scrapeTargets, err := cmd.Flags().GetStringArray("scrape-target")
	if err != nil {
		return err
	}

	if len(scrapeTargets) > 0 {
		opts.obsOpts.scrapeTargets = scrapeTargets
	} else {
		opts.obsOpts.scrapeTargets = defaultScrapeTargets(opts)
	}

	if opts.connectorOpts.allowAllOrigins, err = cmd.Flags().GetBool(
		"connector-allow-all-origins"); err != nil {
		return err
	}

	return nil
}

// parseBackendFlag parses a single --backend spec of the form
// "name=..,host=..,port=..,auth=..". Only name is mandatory; host defaults to
// localhost, port to the default backend port, and auth to false.
func parseBackendFlag(spec string) (backendEntry, error) {
	backend := backendEntry{host: defaultBackendHost, port: defaultBackendPort}

	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		key, val, ok := strings.Cut(part, "=")
		if !ok {
			return backendEntry{}, fmt.Errorf("invalid segment %q: expected key=value", part)
		}

		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)

		switch key {
		case "name":
			backend.name = val
		case "host":
			backend.host = val
		case "port":
			port, err := parsePort(val)
			if err != nil {
				return backendEntry{}, err
			}

			backend.port = port
		case "auth":
			auth, err := strconv.ParseBool(val)
			if err != nil {
				return backendEntry{}, fmt.Errorf("invalid auth %q: %w", val, err)
			}

			backend.auth = auth
		default:
			return backendEntry{}, fmt.Errorf("unknown key %q", key)
		}
	}

	if backend.name == "" {
		return backendEntry{}, errors.New("backend name is required")
	}

	if backend.host == "" {
		backend.host = defaultBackendHost
	}

	return backend, nil
}

// promptBackends collects one or more backend target entries using huh. The echo
// service can be registered as a router backend up front; when present the user
// may finish without registering any manual backend.
func promptBackends(opts *configOptions) error {
	if len(opts.backends) > 0 {
		yes, err := askYesNo("Add more backends besides echo?", "")
		if err != nil {
			return err
		}

		if !yes {
			return nil
		}
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

		yes, err := askYesNo("Add another backend?", "")
		if err != nil {
			return err
		}

		if !yes {
			break
		}
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

					if hasBackend(opts.backends, strings.TrimSpace(s)) {
						return fmt.Errorf("%w: %q", errDuplicateBackend, strings.TrimSpace(s))
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
				Placeholder(defaultBackendHost).
				Value(&host),

			huh.NewInput().
				Title(fmt.Sprintf("Backend %q — port", name)).
				Placeholder(strconv.Itoa(defaultBackendPort)).
				Validate(func(s string) error {
					_, err := parsePort(s)
					return err
				}).
				Value(&portStr),
		),
	)

	if err := detailForm.Run(); err != nil {
		return fmt.Errorf("prompt cancelled: %w", err)
	}

	if strings.TrimSpace(host) == "" {
		host = defaultBackendHost
	}

	port, err := parsePort(portStr)
	if err != nil {
		return err
	}

	opts.backends = append(opts.backends, backendEntry{
		name: name,
		host: host,
		port: port,
	})

	return nil
}

// echoBackend returns the backend entry for the built-in echo service.
func echoBackend() backendEntry {
	return backendEntry{
		name: echoBackendName,
		host: echoBackendHost,
		port: echoBackendPort,
	}
}

// runForm runs a form made of the given groups.
func runForm(groups ...*huh.Group) error {
	if err := huh.NewForm(groups...).Run(); err != nil {
		return fmt.Errorf("prompt cancelled: %w", err)
	}

	return nil
}

// defaultScrapeTargets returns the scrape targets pre-selected by default:
// kerberos, every registered router backend, and jaeger, loki and the
// admin-connector when they are included.
func defaultScrapeTargets(opts *configOptions) []string {
	options := scrapeTargetOptions(opts)

	targets := make([]string, 0, len(options))
	for _, o := range options {
		targets = append(targets, o.Value)
	}

	return targets
}

// scrapeTargetOptions lists the services Prometheus can scrape in this deployment: kerberos,
// every registered router backend, and jaeger, loki and the admin-connector when included.
func scrapeTargetOptions(opts *configOptions) []huh.Option[string] {
	options := []huh.Option[string]{
		huh.NewOption(fmt.Sprintf("kerberos (port %d)", scrapeMetricsPort), defaultKRBDB),
	}

	if opts.includes(jaegerName) {
		options = append(options, huh.NewOption("jaeger (port 8888)", jaegerName))
	}

	if opts.includes(lokiName) {
		options = append(options, huh.NewOption("loki (port 3100)", lokiName))
	}

	for _, b := range opts.backends {
		options = append(options, huh.NewOption(
			fmt.Sprintf("%s (%s:%d)", b.name, b.host, scrapeMetricsPort), b.name))
	}

	if opts.includeConnector {
		options = append(options, huh.NewOption(
			fmt.Sprintf("%s (port %d)", connectorName, scrapeMetricsPort), connectorName))
	}

	return options
}

// buildObsOptionsGroup builds the settings of the chosen observability stack services, or
// returns nil when none of them has settings.
func buildObsOptionsGroup(opts *configOptions) *huh.Group {
	fields := make([]huh.Field, 0, 2)

	if opts.includes(prometheusName) {
		fields = append(fields, huh.NewMultiSelect[string]().
			Title("Prometheus scrape targets").
			Description("Select services that should expose metrics to Prometheus.").
			Options(scrapeTargetOptions(opts)...).
			Value(&opts.obsOpts.scrapeTargets))
	}

	if opts.includes(grafanaName) {
		fields = append(fields, newConfirm(
			"Enable Grafana anonymous access?",
			"Allows viewing dashboards without logging in.",
			&opts.obsOpts.grafanaAnonymous,
		))
	}

	if len(fields) == 0 {
		return nil
	}

	return huh.NewGroup(fields...).Title("Observability stack")
}

// buildConnectorConfigGroup builds the admin-connector settings.
func buildConnectorConfigGroup(opts *configOptions) *huh.Group {
	return huh.NewGroup(
		newConfirm(
			"Allow all CORS origins?",
			"Yes allows any origin; No denies all cross-origin requests.",
			&opts.connectorOpts.allowAllOrigins,
		),
	).Title("Admin-connector")
}

// hasBackend reports whether a backend with the given name is already registered.
func hasBackend(backends []backendEntry, name string) bool {
	for _, b := range backends {
		if b.name == name {
			return true
		}
	}

	return false
}

// parsePort parses a backend port. An empty value means the default port.
func parsePort(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultBackendPort, nil
	}

	port, err := strconv.Atoi(raw)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("%w: %q", errInvalidPort, raw)
	}

	return port, nil
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
