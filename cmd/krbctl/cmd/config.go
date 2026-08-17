package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const (
	driverPostgres = "postgres"
	driverSQLite   = "sqlite"
	defaultKRBDB   = "kerberos"
)

// configOptions holds the answers collected from the interactive config session.
type configOptions struct {
	backends        []backendEntry
	includeAuth     bool
	includeObs      bool
	persistenceMode string // "sqlite" or "postgres"
	outputPath      string
}

type backendEntry struct {
	name string
	host string
	port int
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Interactively generate a base Kerberos configuration file",
		Long: `Walks you through a series of prompts to build a base Kerberos JSON
configuration file. Mandatory sections are always included; optional sections
(auth, observability, postgres persistence) can be skipped.`,
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

	scanner := bufio.NewScanner(os.Stdin)
	opts := &configOptions{outputPath: output}

	fmt.Fprintln(os.Stdout, "=== Kerberos krb.json generator ===")
	fmt.Fprintln(os.Stdout)

	// Backend targets (mandatory — gateway needs at least one)
	opts.backends = promptBackends(scanner)

	// Optional: auth
	opts.includeAuth = promptYesNo(scanner,
		"Include the auth section (basic authentication)? [y/N]")

	// Optional: observability
	opts.includeObs = promptYesNo(scanner,
		"Include the observability section? [y/N]")

	// Optional: postgres persistence
	if promptYesNo(scanner, "Use PostgreSQL as the persistence backend (default: SQLite)? [y/N]") {
		opts.persistenceMode = driverPostgres
	} else {
		opts.persistenceMode = driverSQLite
	}

	content, err := buildConfig(opts)
	if err != nil {
		return fmt.Errorf("failed to build config: %w", err)
	}

	if err := os.WriteFile(opts.outputPath, content, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Fprintf(os.Stdout, "\nkrb.json written to %s\n", opts.outputPath)

	return nil
}

// promptBackends collects one or more backend target entries from the user.
func promptBackends(scanner *bufio.Scanner) []backendEntry {
	fmt.Fprintln(os.Stdout, "Configure gateway backend targets.")
	fmt.Fprintln(
		os.Stdout,
		"(At least one backend is required. Press Enter with an empty name to finish.)",
	)
	fmt.Fprintln(os.Stdout)

	var backends []backendEntry

	for {
		name := promptString(
			scanner,
			fmt.Sprintf("  Backend %d name (e.g. \"my-api\"): ", len(backends)+1),
		)
		if name == "" {
			if len(backends) == 0 {
				fmt.Fprintln(os.Stdout, "  At least one backend is required. Please enter a name.")
				continue
			}

			break
		}

		host := promptString(scanner, "  Host (e.g. \"my-api\" or \"localhost\"): ")
		if host == "" {
			host = "localhost"
		}

		port := parsePort(promptString(scanner, "  Port (e.g. 8080): "))

		backends = append(backends, backendEntry{name: name, host: host, port: port})
		fmt.Fprintln(os.Stdout)

		if !promptYesNo(scanner, "  Add another backend? [y/N]") {
			break
		}

		fmt.Fprintln(os.Stdout)
	}

	return backends
}

func parsePort(raw string) int {
	const defaultPort = 8080

	port, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || port < 1 || port > 65535 {
		fmt.Fprintln(os.Stdout, "  Invalid port, defaulting to 8080.")

		return defaultPort
	}

	return port
}

// promptString prints the question and reads a line from the scanner.
func promptString(scanner *bufio.Scanner, question string) string {
	fmt.Fprint(os.Stdout, question)

	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}

	return ""
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
