package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/kaptinlin/jsonschema"
	"github.com/trebent/zerologr"
)

type (
	Map interface {
		Register(name string, cfg Config)
		Load(name string, data []byte) error
		Parse() error
		Access(name string) (Config, error)
	}
	Config interface {
		Schema() *jsonschema.Schema
	}
	configEntry struct {
		schema *jsonschema.Schema
		cfg    Config
		data   []byte
	}
	impl struct {
		configEntries map[string]*configEntry
	}
)

var (
	//nolint:gochecknoglobals
	NoSchema = &jsonschema.Schema{}

	ErrNoRegisteredName = errors.New("could not find a config entry with that name")
	ErrEnvVarRef        = errors.New("could not find an environment variable")
	ErrUnmarshal        = errors.New("failed to decode configuration")
	ErrSubmatchEnv      = errors.New("failed to find submatch in env match")

	envRe = regexp.MustCompile(`\$\{env:([a-zA-Z0-9_:]+)\}`)
	// pathRe       = regexp.MustCompile(`\$\{ref:([a-zA-Z0-9_:]+)\}`).
	unresolvedRe = regexp.MustCompile(`\$\{UNRESOLVED:([a-zA-Z0-9_]+)\}`)
)

func New() Map {
	return &impl{
		configEntries: make(map[string]*configEntry),
	}
}

func (c *impl) Register(name string, cfg Config) {
	c.configEntries[name] = &configEntry{cfg.Schema(), cfg, nil}
}

func (c *impl) Load(name string, data []byte) error {
	entry, ok := c.configEntries[name]
	if !ok {
		return fmt.Errorf("%w: %s", ErrNoRegisteredName, name)
	}

	entry.data = data

	return nil
}

func (c *impl) Access(name string) (Config, error) {
	entry, ok := c.configEntries[name]

	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNoRegisteredName, name)
	}

	return entry.cfg, nil
}

func (c *impl) Parse() error {
	if err := c.resolveReferences(); err != nil {
		return err
	}

	if err := c.validateSchemas(); err != nil {
		return err
	}

	if err := c.loadData(); err != nil {
		return err
	}

	return nil
}

func (c *impl) resolveReferences() error {
	// Walk across all JSON objects and collect:
	// 1. Env references
	// 2. Path references
	//
	// Resolve env references where possible, replace directly.
	//
	// Resolve path references in the following order:
	// 1. Iterate over all path references
	// 2. Go to the referenced value
	//    - If the referenced value is another reference, go to the referenced value etc. until a value is found.
	//      If there are only references, return an error.

	errChan := make(chan error, len(c.configEntries))

	for name, entry := range c.configEntries {
		zerologr.V(5).Info("Running environment variable resolution for config " + name)
		go c.resolveEnvironmentVariables(name, entry, errChan)
	}

	errs := make([]error, 0, len(c.configEntries))
	for {
		errs = append(errs, <-errChan)

		if len(errs) == len(c.configEntries) {
			break
		}
	}

	if err := errors.Join(errs...); err != nil {
		zerologr.Error(err, "Error(s) occurred during environment variable resolution")
		return err
	}

	for name := range c.configEntries {
		zerologr.V(5).Info("Running path variable resolution for config " + name)
	}

	return nil
}

func (c *impl) resolveEnvironmentVariables(name string, entry *configEntry, errChan chan error) {
	logger := zerologr.WithValues("config_name", name)

	defer func() {
		if r := recover(); r != nil {
			logger.Error(
				fmt.Errorf("%w: panic while processing config %s: %v", ErrEnvVarRef, name, r),
				"Panic during env var resolution",
			)
			errChan <- fmt.Errorf("%w: panic while processing config %s", ErrEnvVarRef, name)
		}
	}()

	entry.data = envRe.ReplaceAllFunc(entry.data, func(match []byte) []byte {
		groups := envRe.FindSubmatch(match)
		if len(groups) != 2 {
			logger.Error(
				fmt.Errorf(
					"%w: failed to find submatch for env replacement while processing config %s",
					ErrSubmatchEnv,
					name,
				),
				"Failed to find submatch in config",
			)
			return match
		}

		parts := strings.Split(string(groups[1]), ":")
		val, ok := os.LookupEnv(parts[0])
		if !ok && len(parts) > 1 {
			val = parts[1]
		} else if !ok {
			return fmt.Appendf([]byte{}, "${UNRESOLVED:%s}", parts[0])
		}

		return []byte(val)
	})

	matches := unresolvedRe.FindAll(entry.data, -1)
	if len(matches) > 0 {
		logger.Error(ErrEnvVarRef, "Failed to find some environment variable(s)")
		errMsg := "failed to find environment variables: "

		var builder strings.Builder
		for _, match := range matches {
			logger.V(5).Info("Unresolved env var match: " + string(match))
			groups := unresolvedRe.FindSubmatch(match)
			builder.WriteString(string(groups[1]) + ", ")
		}
		errMsg += builder.String()
		errMsg = strings.TrimSuffix(errMsg, ", ")

		errChan <- fmt.Errorf("%w: %s", ErrEnvVarRef, errMsg)
		return
	}

	logger.V(100).Info("Replaced environment variables: " + string(entry.data))

	errChan <- nil
}

func (c *impl) validateSchemas() error {
	return nil
}

func (c *impl) loadData() error {
	for name, entry := range c.configEntries {
		if len(entry.data) == 0 {
			continue
		}

		if err := json.Unmarshal(entry.data, entry.cfg); err != nil {
			return fmt.Errorf("%w: %s due to: %w", ErrUnmarshal, name, err)
		}
	}

	return nil
}
