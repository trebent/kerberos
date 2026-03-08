package config

import (
	"bytes"
	_ "embed"
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
	RootConfig struct {
		data        []byte
		escapedData []byte

		values map[string]any
		refs   map[string]string
	}
)

var (
	envRe  = regexp.MustCompile(`\$\{env:([a-zA-Z0-9_:]+)\}`)
	pathRe = regexp.MustCompile(`\$\{ref:([a-zA-Z0-9_\.\[\]:]+)\}`)

	//go:embed schemas/ordered_schema.json
	schemaBytesOrdered []byte
	//go:embed schemas/admin_schema.json
	schemaBytesAdmin []byte
	//go:embed schemas/auth_schema.json
	schemaBytesAuth []byte
	//go:embed schemas/observability_schema.json
	schemaBytesObservability []byte
	//go:embed schemas/router_schema.json
	schemaBytesRouter []byte
	//go:embed schemas/oas_schema.json
	schemaBytesOAS []byte
	//go:embed schemas/config_schema.json
	schemaBytesConfig []byte
)

func New() *RootConfig {
	return &RootConfig{
		values: make(map[string]any),
		refs:   make(map[string]string),
	}
}

func (c *RootConfig) Load(data []byte) {
	c.data = data
}

func (c *RootConfig) Parse() error {
	if err := c.resolveReferences(); err != nil {
		return err
	}

	if err := c.validateSchema(); err != nil {
		return err
	}

	if err := c.loadData(); err != nil {
		return err
	}

	// Free allocated memory for intermediate data structures.
	c.data = nil
	c.escapedData = nil
	c.values = nil
	c.refs = nil

	return nil
}

func (c *RootConfig) resolveReferences() error {
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

func (c *RootConfig) escapeReferences() error {
	zerologr.V(100).Info("Escaping references")
	c.escapedData = c.data

	i := 0
	for i < len(c.data) {
		//nolint:gocritic // ignore: nestedIfs
		if c.data[i] == '$' && isReference(c.data[i:i+6]) && c.data[i-1] != '"' {
			zerologr.V(100).Info(
				fmt.Sprintf("Found unescaped reference at index %d", i),
			)

			end := bytes.IndexByte(c.data[i:], '}')
			if end == -1 {
				return errors.New("malformed reference: missing closing '}'")
			}

			escapedRef := append([]byte{'"'}, c.data[i:i+end+1]...)
			escapedRef = append(escapedRef, '"')

			zerologr.V(100).Info("Escaped reference: " + string(escapedRef))

			c.escapedData = bytes.Replace(c.escapedData, c.data[i:i+end+1], escapedRef, 1)

			zerologr.V(100).Info("Intermediate escaped data: \n" + string(c.escapedData))

			i = i + end + 3
		} else if c.data[i] == '$' && isReference(c.data[i:i+6]) {
			zerologr.V(100).Info(
				fmt.Sprintf("Found already escaped reference at index %d", i),
			)

			end := bytes.IndexByte(c.data[i:], '}')
			if end == -1 {
				return errors.New("malformed reference: missing closing '}'")
			}

			i = i + end + 2
		} else {
			i++
		}
	}

	zerologr.V(100).Info("Escaped references")

	return nil
}

func (c *RootConfig) walkForReferences() error {
	if len(c.data) == 0 {
		return nil
	}

	zerologr.V(100).Info("Gathering references")

	generic := make(map[string]any)
	if err := json.Unmarshal(c.escapedData, &generic); err != nil {
		return err
	}

	if err := c.walk("", generic); err != nil {
		return err
	}

	zerologr.V(100).Info("Gathered references")
	zerologr.V(100).Info(fmt.Sprintf("Current values map: %+v", c.values))
	zerologr.V(100).Info(fmt.Sprintf("Current refs map: %+v", c.refs))

	return nil
}

func (c *RootConfig) walk(currentPath string, generic any) error {
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

			newPath := currentPath
			if newPath != "" {
				newPath += "."
			}
			newPath += k

			if err := c.walk(newPath, v); err != nil {
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

func (c *RootConfig) findReferenceValues() error {
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
			zerologr.V(100).Info("Resolving environment reference value for: " + ref)
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
			zerologr.V(100).Info("Resolving path reference value for: " + ref)

			// Find if the path reference can be walked to a final value
			c.refs[ref], err = c.findReferenceValue(ref)
			if err != nil {
				return err
			}

			zerologr.V(100).Info("Resolved path reference value for: " + ref + ", value: " + c.refs[ref])
		}
	}

	zerologr.V(100).Info(fmt.Sprintf("Realised all refs: %s", c.refs))

	return nil
}

func (c *RootConfig) findReferenceValue(ref string) (string, error) {
	zerologr.V(100).Info("Finding value for path reference: " + ref)

	valuePath, err := getPathFromReference(ref)
	if err != nil {
		return "", err
	}

	value, ok := c.values[valuePath]
	if !ok {
		return "", errors.New("referenced path was not found: " + valuePath)
	}

	decoded, ok := value.(string)
	if ok {
		if isEnvReference(decoded) {
			zerologr.V(100).Info("Nested environment reference found: " + decoded)
			return c.refs[decoded], nil
		} else if isPathReference(decoded) {
			zerologr.V(100).Info("Nested path reference found, walking path: " + decoded)
			return c.walkRefs(valuePath, decoded)
		}
	}

	return fmt.Sprint(value), nil
}

func (c *RootConfig) walkRefs(originPath, ref string) (string, error) {
	zerologr.V(100).Info("Walking references, origin: " + originPath + ", ref: " + ref)

	valuePath, err := getPathFromReference(ref)
	if err != nil {
		return "", err
	}

	if originPath == valuePath {
		return "", errors.New("circular reference detected: " + originPath)
	}

	value, ok := c.values[valuePath]
	if !ok {
		return "", errors.New("reference path not found: " + valuePath)
	}

	decoded, ok := value.(string)
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
	zerologr.V(100).Info("Final value found for " + originPath + " during walk: " + fmt.Sprint(value))

	return fmt.Sprint(value), nil
}

func (c *RootConfig) replaceReferencesInData() error {
	/*
		Replace all references in the original JSON data with their resolved values.
	*/
	if len(c.data) == 0 {
		return nil
	}

	dataStr := string(c.data)
	zerologr.V(100).Info("Replacing references: " + dataStr)

	for ref, val := range c.refs {
		zerologr.V(100).Info(
			fmt.Sprintf("Replacing reference '%s' with value '%s'", ref, val),
		)
		dataStr = strings.ReplaceAll(dataStr, ref, val)
		zerologr.V(100).Info("Intermediate replaced data: " + dataStr)
	}

	c.data = []byte(dataStr)

	zerologr.V(100).Info("Replaced all references: " + string(c.data))

	return nil
}

func (c *RootConfig) validateSchema() error {
	zerologr.V(100).Info("Validating schema")

	if len(c.data) == 0 {
		return nil
	}

	// Since the root validation schema is registered anonymously, we need to compile it here, per
	// configuration entry.
	sl := gojsonschema.NewSchemaLoader()
	sl.AutoDetect = false
	sl.Validate = true
	sl.Draft = gojsonschema.Draft7
	if err := sl.AddSchemas(
		gojsonschema.NewBytesLoader(schemaBytesOrdered),
		gojsonschema.NewBytesLoader(schemaBytesAdmin),
		gojsonschema.NewBytesLoader(schemaBytesAuth),
		gojsonschema.NewBytesLoader(schemaBytesObservability),
		gojsonschema.NewBytesLoader(schemaBytesRouter),
		gojsonschema.NewBytesLoader(schemaBytesOAS),
	); err != nil {
		zerologr.Error(err, "Failed to add global schemas")
		return err
	}

	compiledSchema, err := sl.Compile(gojsonschema.NewBytesLoader(schemaBytesConfig))
	if err != nil {
		zerologr.Error(err, "Failed to compile root schema")
		return err
	}

	result, err := compiledSchema.Validate(gojsonschema.NewBytesLoader(c.data))
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

		return fmt.Errorf("schema validation failed: %w", fullError)
	}

	return nil
}

func (c *RootConfig) loadData() error {
	return json.Unmarshal(c.data, c)
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
		return "", errors.New("malformed env var reference: " + ref)
	}

	split := strings.Split(groups[1], ":")

	val, ok := os.LookupEnv(split[0])
	if !ok {
		if len(split) > 1 {
			zerologr.V(100).Info("Using default env var value: " + split[1])
			return split[1], nil
		}
		return "", errors.New("environment variable not found: " + split[0])
	}
	zerologr.V(100).Info("Found env var value: " + val)

	return val, nil
}

func getPathFromReference(ref string) (string, error) {
	groups := pathRe.FindStringSubmatch(ref)
	if len(groups) < 2 {
		return "", errors.New("malformed path reference: " + ref)
	}
	zerologr.V(100).Info("Extracted path from reference", "ref", ref, "path", groups[1])

	return groups[1], nil
}
