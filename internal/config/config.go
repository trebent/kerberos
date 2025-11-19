package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
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
		// values config values
		values map[string]any
		refs   map[string]string
	}
)

var (
	//nolint:gochecknoglobals
	NoSchema = &jsonschema.Schema{}

	ErrNoRegisteredName = errors.New("could not find a config entry with that name")
	ErrEnvVarRef        = errors.New("could not find an environment variable")
	ErrPathVarRef       = errors.New("could not find path variable")

	ErrMalformedPathRef = errors.New("malformed path reference")
	ErrMalformedEnvRef  = errors.New("malformed env reference")
	ErrUnmarshal        = errors.New("failed to decode configuration")
	ErrSubmatchEnv      = errors.New("failed to find submatch in env match")

	envRe  = regexp.MustCompile(`\$\{env:([a-zA-Z0-9_:]+)\}`)
	pathRe = regexp.MustCompile(`\$\{ref:([a-zA-Z0-9_\.\[\]:]+)\}`)
	// unresolvedRe = regexp.MustCompile(`\$\{UNRESOLVED:([a-zA-Z0-9_]+)\}`)
)

func New() Map {
	return &impl{
		configEntries: make(map[string]*configEntry),
		values:        make(map[string]any),
		refs:          make(map[string]string),
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

	if err := c.escapeReferences(); err != nil {
		return err
	}

	if err := c.walkConfig(); err != nil {
		return err
	}

	if err := c.realiseReferences(); err != nil {
		return err
	}

	return nil
}

func (c *impl) escapeReferences() error {
	/*
		Reads through all config entries and escapes any references found to prevent them from being
		interpreted incorrectly during JSON unmarshalling.
	*/
	for name, entry := range c.configEntries {
		zerologr.V(100).Info("Escaping references for config '" + name + "'")

		i := 0
		for i < len(entry.data) {
			println(strconv.Itoa(i) + ": " + string(entry.data[i]))

			if entry.data[i] == '$' && isReference(entry.data[i:i+6]) && entry.data[i-1] != '"' {
				zerologr.V(100).Info("Found unescaped reference at index " + fmt.Sprint(i) + " in config '" + name + "'")

				end := bytes.IndexByte(entry.data[i:], '}')
				if end == -1 {
					return fmt.Errorf("%w: %s", ErrMalformedPathRef, name)
				}

				escapedRef := append([]byte{'"'}, entry.data[i:i+end+1]...)
				escapedRef = append(escapedRef, '"')

				zerologr.V(100).Info("Escaped reference: " + string(escapedRef))

				entry.data = bytes.Replace(entry.data, entry.data[i:i+end+1], escapedRef, 1)

				i = i + end + 3
			} else if entry.data[i] == '$' && isReference(entry.data[i:i+6]) {
				zerologr.V(100).Info("Found already escaped reference at index " + fmt.Sprint(i) + " in config '" + name + "'")

				end := bytes.IndexByte(entry.data[i:], '}')
				if end == -1 {
					return fmt.Errorf("%w: %s", ErrMalformedPathRef, name)
				}

				i = i + end + 2
			} else {
				i++
			}
		}

		zerologr.V(100).Info("Escaped references for config '" + name + "': " + string(entry.data))
	}

	return nil
}

func (c *impl) walkConfig() error {
	/*
		Reads through all config entries and gathers references and values for later resolution.
	*/
	for name, entry := range c.configEntries {
		zerologr.V(100).Info("Gathering references for config '" + name + "'")

		generic := make(map[string]any)
		if err := json.Unmarshal(entry.data, &generic); err != nil {
			return fmt.Errorf("%w: %s due to: %w", ErrUnmarshal, name, err)
		}

		if err := c.walk(name, generic); err != nil {
			return err
		}

		zerologr.V(100).Info("Gathered references for config '" + name + "'")
		zerologr.V(100).Info(fmt.Sprintf("Current values map: %+v", c.values))
		zerologr.V(100).Info(fmt.Sprintf("Current refs map: %+v", c.refs))
	}

	return nil
}

func (c *impl) walk(currentPath string, generic any) error {
	/*
		Walk through a JSON object and find final values for all JSON paths.

		For each found value, store the path and the value it contains.
		For each found reference, store the reference path.
		Enabled to replace the reference path with the actual value.
	*/
	zerologr.V(100).Info("Walking for references in path '" + currentPath + "'")

	switch val := generic.(type) {
	case map[string]any:
		zerologr.V(100).Info("Walking into map in path '" + currentPath + "'")
		for k, v := range val {
			zerologr.V(100).Info("Key: " + k + ", Value: " + fmt.Sprint(v))
			if err := c.walk(currentPath+"."+k, v); err != nil {
				return err
			}
		}
	case []any:
		zerologr.V(100).Info("Walking into array in path '" + currentPath + "'")
		for i, item := range val {
			if err := c.walk(currentPath+".["+strconv.Itoa(i)+"]", item); err != nil {
				return err
			}
		}
	case string:
		if isReference([]byte(val)) {
			zerologr.V(100).Info("Found ref: " + val)
			c.refs[val] = ""
		}
		c.values[currentPath] = val
	default:
		zerologr.V(100).Info("Storing final value for path '" + currentPath + "'")
		c.values[currentPath] = val
	}

	return nil
}

func (c *impl) realiseReferences() error {
	/*
		Values contain values for full paths, but some contains references as well. Now what's needed is:

		For values for a given path: return the value if it's not a reference.
		For references: find if the reference can be walked to a value. Environment variables can be resolved
		immediately, path references need to be walked through the values map until a final value is found.

		Environment variable references are resolved first, then path references.
	*/
	for ref := range c.refs {
		zerologr.V(100).Info("Checking reference: " + ref)
		var err error
		if isEnvReference(ref) {
			zerologr.V(100).Info("Env ref, looking up environment variable: " + ref)
			c.refs[ref], err = getEnvReference(ref)
			if err != nil {
				return err
			}
		}
	}

	zerologr.V(100).Info(fmt.Sprintf("Realised env references: %s", c.refs))

	for ref := range c.refs {
		zerologr.V(100).Info("Checking reference: " + ref)
		var err error
		if isPathReference(ref) {
			zerologr.V(100).Info("Path ref, finding final value: " + ref)

			// Find if the path reference can be walked to a final value
			c.refs[ref], err = c.realiseReference(ref)
			if err != nil {
				return err
			}
		}
	}

	zerologr.V(100).Info(fmt.Sprintf("Realised references: %s", c.refs))

	return nil
}

func (c *impl) realiseReference(origin string) (string, error) {
	zerologr.V(100).Info("Walking refs, origin: " + origin)

	originPath, err := c.getPathFromReference(origin)
	if err != nil {
		return "", err
	}

	value, ok := c.values[originPath]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrPathVarRef, origin)
	}

	switch decoded := value.(type) {
	case string:
		if isEnvReference(decoded) {
			zerologr.V(100).Info("Env ref found during walk: " + decoded)
			return c.refs[decoded], nil
		} else if isPathReference(decoded) {
			ref, err := c.getPathFromReference(decoded)
			if err != nil {
				return "", err
			}

			return c.walkRefs(originPath, ref)
		}
	}
	return fmt.Sprintf("%v", value), nil
}

func (c *impl) walkRefs(originPath, ref string) (string, error) {
	zerologr.V(100).Info("Walking refs, origin: " + originPath + ", ref: " + ref)

	newRefPath, err := c.getPathFromReference(ref)
	if err != nil {
		return "", err
	}

	if originPath == newRefPath {
		return "", fmt.Errorf("%w: circular reference detected at %s", ErrPathVarRef, originPath)
	}

	val, ok := c.values[newRefPath]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrPathVarRef, ref)
	}

	switch decoded := val.(type) {
	case string:
		if originPath == decoded {
			return "", fmt.Errorf("%w: circular reference detected at %s", ErrPathVarRef, originPath)
		} else if isPathReference(decoded) {
			zerologr.V(100).Info("Path ref found during walk: " + decoded)
			newRef, err := c.getPathFromReference(decoded)
			if err != nil {
				return "", err
			}
			return c.walkRefs(originPath, newRef)
		}
	default:
		return fmt.Sprintf("%v", decoded), nil
	}

	return fmt.Sprintf("%v", val), nil
}

func (c *impl) getPathFromReference(ref string) (string, error) {
	groups := pathRe.FindStringSubmatch(ref)
	zerologr.V(100).Info("Found path ref submatch groups: ", "ref", ref, "groups", groups)
	if len(groups) < 2 {
		return "", fmt.Errorf("%w: %s", ErrMalformedPathRef, ref)
	}

	split := strings.Split(groups[1], ":")
	return split[0], nil
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

func isReference(data []byte) bool {
	zerologr.V(100).Info("isReference check on: " + string(data))

	prefixes := [][]byte{[]byte("${env:"), []byte("${ref:")}

	for _, prefix := range prefixes {
		if bytes.HasPrefix(data, prefix) {
			return true
		}
	}

	return false
}

func isEnvReference(ref string) bool {
	return strings.HasPrefix(ref, "${env:")
}

func isPathReference(ref string) bool {
	return strings.HasPrefix(ref, "${ref:")
}

func getEnvReference(ref string) (string, error) {
	groups := envRe.FindStringSubmatch(ref)
	if len(groups) < 2 {
		return "", fmt.Errorf("%w: %s", ErrMalformedEnvRef, ref)
	}

	split := strings.Split(groups[1], ":")

	val, ok := os.LookupEnv(split[0])
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrEnvVarRef, split[0])
	}

	return val, nil
}
