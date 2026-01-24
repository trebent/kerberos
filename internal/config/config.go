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

	"github.com/trebent/zerologr"
	"github.com/xeipuuv/gojsonschema"
)

type (
	// Map is a configuration map that holds multiple configuration entries.
	//
	// Each configuration entry is registered with a name and a config struct that implements the Config interface.
	//
	// Configuration data can be loaded for each entry, and all entries can be parsed to resolve references,
	// validate against schemas, and unmarshal into the config structs.
	//
	// Example usage:
	//   cfgMap := config.New(&config.Opts{})
	//   cfgMap.Register("myConfig", &MyConfigStruct{})
	//   err := cfgMap.Load("myConfig", []byte(`{"key": "value"}`))
	//   if err != nil {
	//       panic(err)
	//   }
	//   err = cfgMap.Parse()
	//   if err != nil {
	//       panic(err)
	//   }
	//   myCfg, err := cfgMap.Access("myConfig")
	//   if err != nil {
	//       panic(err)
	//   }
	//
	// This creates a configuration map, registers a config entry, loads JSON data into it,
	// parses all entries, and accesses the populated config struct.
	//
	// Returns errors if any step fails, such as loading data for an unregistered name,
	// schema validation failures, or unmarshalling errors.
	Map interface {
		// Register a configuration producer entry with a name and a config struct.
		// The config struct must implement the Config interface.
		//
		// Example:
		//   cfgMap.Register("myConfig", &MyConfigStruct{})
		//
		// This registers a configuration entry named "myConfig" with the provided config struct.
		//
		// The config struct will be populated when Load and Parse are called.
		//
		// Panics if the name is already registered.
		Register(name string, cfg Config)
		// Load configuration data for a registered config entry by name.
		//
		// Example:
		//   err := cfgMap.Load("myConfig", []byte(`{"key": "value"}`))
		//
		// This loads the provided JSON data into the config entry named "myConfig".
		// Returns an error if the name is not registered.
		//
		// MustLoad is similar to Load but panics on error.
		//
		// The actual unmarshalling into the config struct happens when Parse is called.
		//
		// Returns an error if the name is not registered.
		Load(name string, data []byte) error
		// MustLoad is similar to Load but panics on error.
		//
		// Example:
		//   cfgMap.MustLoad("myConfig", []byte(`{"key": "value"}`))
		//
		// This loads the provided JSON data into the config entry named "myConfig".
		// Panics if the name is not registered.
		//
		// The actual unmarshalling into the config struct happens when Parse is called.
		//
		// Panics if the name is not registered.
		MustLoad(name string, data []byte)
		// Parse all loaded configuration entries.
		//
		// This performs the following steps:
		// 1. Resolves all environment variable and path references in the loaded JSON data.
		// 2. Validates the JSON data against the schema provided by the config struct.
		// 3. Unmarshals the JSON data into the config struct.
		//
		// Returns an error if any step fails.
		Parse() error
		// Access a configuration entry by name.
		//
		// Example:
		//   cfg, err := cfgMap.Access("myConfig")
		//
		// This accesses the config entry named "myConfig" and returns the config struct.
		//
		// Returns an error if the name is not registered.
		Access(name string) (Config, error)
	}
	Config interface {
		// SchemaJSONLoader returns the JSON loader for the config struct schema.
		//
		// Example:
		//   func (c *MyConfigStruct) SchemaJSONLoader() gojsonschema.JSONLoader {
		//       return gojsonschema.NewBytesLoader([]byte(`{
		//           "type": "object",
		//           "properties": {
		//               "key": { "type": "string" }
		//           },
		//           "required": ["key"]
		//       }`))
		//   }
		//
		// This provides the JSON loader for the schema used in validation.
		//
		// Returns nil if no schema validation is needed.
		SchemaJSONLoader() gojsonschema.JSONLoader
	}
	Opts struct {
		// GlobalSchemas are JSON schemas that are applied to all config entries.
		//
		// These schemas can define common structures or constraints that apply to all config entries.
		//
		// Example:
		//   globalSchemaLoader := gojsonschema.NewBytesLoader([]byte(`{
		//       "type": "object",
		//       "properties": {
		//           "globalKey": { "type": "string" }
		//       }
		//   }`))
		//   opts := &Opts{
		//       GlobalSchemas: []gojsonschema.JSONLoader{globalSchemaLoader},
		//   }
		//
		// This sets a global schema that requires a "globalKey" property in all config entries.
		//
		// If no global schemas are needed, this can be left nil or empty.
		GlobalSchemas []gojsonschema.JSONLoader
	}
	configEntry struct {
		cfg         Config
		data        []byte
		escapedData []byte
	}
	impl struct {
		globalSchemas []gojsonschema.JSONLoader
		configEntries map[string]*configEntry
		// values config values
		values map[string]any
		refs   map[string]string
	}
)

var (
	//nolint:gochecknoglobals
	NoSchema = &gojsonschema.Schema{}

	ErrNoRegisteredName   = errors.New("could not find a config entry with that name")
	ErrEnvVarRef          = errors.New("could not find an environment variable")
	ErrPathVarRef         = errors.New("could not find path variable")
	ErrPathVarRefCircular = errors.New("circular path reference detected")
	ErrMalformedPathRef   = errors.New("malformed path reference")
	ErrMalformedEnvRef    = errors.New("malformed env reference")
	ErrUnmarshal          = errors.New("failed to decode configuration")
	ErrSubmatchEnv        = errors.New("failed to find submatch in env match")
	ErrSchema             = errors.New("schema validation failed")

	envRe  = regexp.MustCompile(`\$\{env:([a-zA-Z0-9_:]+)\}`)
	pathRe = regexp.MustCompile(`\$\{ref:([a-zA-Z0-9_\.\[\]:]+)\}`)
)

func New(opts *Opts) Map {
	return &impl{
		globalSchemas: opts.GlobalSchemas,
		configEntries: make(map[string]*configEntry),
		values:        make(map[string]any),
		refs:          make(map[string]string),
	}
}

// AccessAs accesses a configuration entry by name and casts it to the provided type T.
// It panics if the config entry is not found or if the type assertion fails.
func AccessAs[T any](cfg Map, name string) T {
	d, err := cfg.Access(name)
	if err != nil {
		panic(err)
	}

	typed, ok := d.(T)
	if !ok {
		panic(fmt.Sprintf("invalid config type for: %s", name))
	}

	return typed
}

func (c *impl) Register(name string, cfg Config) {
	c.configEntries[name] = &configEntry{cfg, nil, nil}
}

func (c *impl) Load(name string, data []byte) error {
	entry, ok := c.configEntries[name]
	if !ok {
		return fmt.Errorf("%w: %s", ErrNoRegisteredName, name)
	}

	entry.data = data

	return nil
}

func (c *impl) MustLoad(name string, data []byte) {
	entry, ok := c.configEntries[name]
	if !ok {
		panic(fmt.Errorf("%w: %s", ErrNoRegisteredName, name))
	}

	entry.data = data
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

	if err := c.walkForReferences(); err != nil {
		return err
	}

	if err := c.findReferenceValues(); err != nil {
		return err
	}

	if err := c.replaceReferencesInData(); err != nil {
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
		if len(entry.data) == 0 {
			continue
		}

		zerologr.V(100).Info("Escaping references for config '" + name + "'")
		entry.escapedData = entry.data

		i := 0
		for i < len(entry.data) {
			//nolint:gocritic // ignore: nestedIfs
			if entry.data[i] == '$' && isReference(entry.data[i:i+6]) && entry.data[i-1] != '"' {
				zerologr.V(100).Info(
					fmt.Sprintf("Found unescaped reference at index %d in config %s", i, name),
				)

				end := bytes.IndexByte(entry.data[i:], '}')
				if end == -1 {
					return fmt.Errorf("%w: %s", ErrMalformedPathRef, name)
				}

				escapedRef := append([]byte{'"'}, entry.data[i:i+end+1]...)
				escapedRef = append(escapedRef, '"')

				zerologr.V(100).Info("Escaped reference: " + string(escapedRef))

				entry.escapedData = bytes.Replace(entry.data, entry.data[i:i+end+1], escapedRef, 1)

				i = i + end + 3
			} else if entry.data[i] == '$' && isReference(entry.data[i:i+6]) {
				zerologr.V(100).Info(
					fmt.Sprintf("Found already escaped reference at index %d in config %s", i, name),
				)

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

func (c *impl) walkForReferences() error {
	/*
		Reads through all config entries and gathers references and values for later resolution.
	*/
	for name, entry := range c.configEntries {
		if len(entry.data) == 0 {
			continue
		}

		zerologr.V(100).Info("Gathering references for config '" + name + "'")

		generic := make(map[string]any)
		if err := json.Unmarshal(entry.escapedData, &generic); err != nil {
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
			if err := c.walk(currentPath+"["+strconv.Itoa(i)+"]", item); err != nil {
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

func (c *impl) findReferenceValues() error {
	zerologr.V(100).Info("Finding reference values")
	/*
		Values contain values for full paths, but some contains references as well. Now what's needed is:

		For values for a given path: return the value if it's not a reference.
		For references: find if the reference can be walked to a value. Environment variables can be resolved
		immediately, path references need to be walked through the values map until a final value is found.

		Environment variable references are resolved first, then path references.
	*/
	for ref := range c.refs {
		var err error
		if isEnvReference(ref) {
			zerologr.V(100).Info("Checking env ref: " + ref)
			c.refs[ref], err = getEnvReferenceValue(ref)
			if err != nil {
				return err
			}
		}
	}

	zerologr.V(100).Info(fmt.Sprintf("Realised env refs: %s", c.refs))

	for ref := range c.refs {
		var err error
		if isPathReference(ref) {
			zerologr.V(100).Info("Checking path ref: " + ref)

			// Find if the path reference can be walked to a final value
			c.refs[ref], err = c.findReferenceValue(ref)
			if err != nil {
				return err
			}
		}
	}

	zerologr.V(100).Info(fmt.Sprintf("Realised all refs: %s", c.refs))

	return nil
}

func (c *impl) findReferenceValue(origin string) (string, error) {
	zerologr.V(100).Info("Finding value for ref: " + origin)

	originPath, err := getPathFromReference(origin)
	if err != nil {
		return "", err
	}

	value, ok := c.values[originPath]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrPathVarRef, origin)
	}

	decoded, ok := value.(string)
	if ok {
		if isEnvReference(decoded) {
			zerologr.V(100).Info("Env ref found: " + decoded)
			return c.refs[decoded], nil
		} else if isPathReference(decoded) {
			zerologr.V(100).Info("Path ref found, walking path: " + decoded)
			return c.walkRefs(originPath, decoded)
		}
	}

	return fmt.Sprintf("%v", value), nil
}

func (c *impl) walkRefs(originPath, ref string) (string, error) {
	zerologr.V(100).Info("Walking refs, origin: " + originPath + ", ref: " + ref)

	newRefPath, err := getPathFromReference(ref)
	if err != nil {
		return "", err
	}
	zerologr.V(100).Info("New ref path: " + newRefPath + " origin: " + originPath)

	if originPath == newRefPath {
		return "", fmt.Errorf("%w: %s", ErrPathVarRefCircular, originPath)
	}

	val, ok := c.values[newRefPath]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrPathVarRef, ref)
	}

	decoded, ok := val.(string)
	if ok {
		if isEnvReference(decoded) {
			zerologr.V(100).Info("Env ref found during walk: " + decoded)
			return c.refs[decoded], nil
		} else if isPathReference(decoded) {
			zerologr.V(100).Info("Path ref found during walk: " + decoded)
			// Don't decode the reference path here, it's done recursively in the next call to walkRefs.
			// The next call to walkRefs will extract the path from the reference and compare it to the originPath.
			return c.walkRefs(originPath, decoded)
		}
	}

	return decoded, nil
}

func (c *impl) replaceReferencesInData() error {
	/*
		Replace all references in the original JSON data with their resolved values.
	*/
	for name, entry := range c.configEntries {
		if len(entry.data) == 0 {
			continue
		}

		zerologr.V(100).Info("Replacing references in config '" + name + "': " + string(entry.data))

		dataStr := string(entry.data)

		for ref, val := range c.refs {
			zerologr.V(100).Info(
				fmt.Sprintf("Replacing reference '%s' with value '%s' in config '%s'", ref, val, name),
			)
			dataStr = strings.ReplaceAll(dataStr, ref, val)
			zerologr.V(100).Info("Intermediate replaced data: " + dataStr)
		}

		entry.data = []byte(dataStr)

		zerologr.V(100).Info("Replaced references in config '" + name + "': " + string(entry.data))
	}

	return nil
}

func (c *impl) validateSchemas() error {
	zerologr.V(100).Info("Validating schemas for all config entries")

	sl := gojsonschema.NewSchemaLoader()
	sl.AutoDetect = false
	sl.Validate = true
	sl.Draft = gojsonschema.Draft7
	sl.AddSchemas(c.globalSchemas...)

	/*
		Validate all loaded config entries against their schemas.
	*/
	for name, entry := range c.configEntries {
		if len(entry.data) == 0 {
			continue
		}

		zerologr.V(100).Info("Validating schema for config entry " + name)
		if entry.cfg.SchemaJSONLoader() == nil {
			zerologr.V(100).Info("No schema defined, skipping validation")
			continue
		}

		compiledSchema, err := sl.Compile(entry.cfg.SchemaJSONLoader())
		if err != nil {
			return err
		}

		result, err := compiledSchema.Validate(gojsonschema.NewBytesLoader(entry.data))
		if err != nil {
			return err
		}

		if !result.Valid() {
			var fullError error
			for _, validationErr := range result.Errors() {
				fullError = fmt.Errorf(
					"%w, %s - %s",
					fullError,
					validationErr.Field(),
					validationErr.Description(),
				)
			}

			return fmt.Errorf(
				"%s: %w: %s",
				name,
				ErrSchema,
				strings.TrimPrefix(fullError.Error(), "<nil>, "),
			)
		}

		zerologr.V(100).Info("Schema for config entry " + name + " is valid")
	}

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

func getEnvReferenceValue(ref string) (string, error) {
	groups := envRe.FindStringSubmatch(ref)
	zerologr.V(100).Info("Found env ref submatch groups: ", "ref", ref, "groups", groups)
	if len(groups) < 2 {
		return "", fmt.Errorf("%w: %s", ErrMalformedEnvRef, ref)
	}

	split := strings.Split(groups[1], ":")

	val, ok := os.LookupEnv(split[0])
	if !ok {
		if len(split) > 1 {
			zerologr.V(100).Info("Using default env var value: " + split[1])
			return split[1], nil
		}
		return "", fmt.Errorf("%w: %s", ErrEnvVarRef, split[0])
	}
	zerologr.V(100).Info("Found env var value: " + val)

	return val, nil
}

func getPathFromReference(ref string) (string, error) {
	groups := pathRe.FindStringSubmatch(ref)
	zerologr.V(100).Info("Found path ref submatch groups: ", "ref", ref, "groups", groups)
	if len(groups) < 2 {
		return "", fmt.Errorf("%w: %s", ErrMalformedPathRef, ref)
	}

	return groups[1], nil
}
