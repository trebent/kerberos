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

		*ObservabilityConfig `json:"observability"`
		*RouterConfig        `json:"router"`
		*AdminConfig         `json:"admin"`
		*OASConfig           `json:"oas,omitempty"`
		*AuthConfig          `json:"auth,omitempty"`
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

func (rc *RootConfig) AuthEnabled() bool {
	return rc.AuthConfig != nil
}

func (rc *RootConfig) OASEnabled() bool {
	return rc.OASConfig != nil
}

func New() *RootConfig {
	return &RootConfig{
		values: make(map[string]any),
		refs:   make(map[string]string),

		ObservabilityConfig: newObservabilityConfig(),
		AdminConfig:         newAdminConfig(),
	}
}

func (rc *RootConfig) Load(data []byte) {
	rc.data = data
}

func (rc *RootConfig) Parse() error {
	if err := rc.resolveReferences(); err != nil {
		return err
	}

	if err := rc.validateSchema(); err != nil {
		return err
	}

	if err := rc.loadData(); err != nil {
		return err
	}

	// Free allocated memory for intermediate data structures.
	rc.data = nil
	rc.escapedData = nil
	rc.values = nil
	rc.refs = nil

	rc.postProcess()

	return nil
}

func (rc *RootConfig) resolveReferences() error {
	if err := rc.escapeReferences(); err != nil {
		return err
	}

	if err := rc.walkForReferences(); err != nil {
		return err
	}

	if err := rc.findReferenceValues(); err != nil {
		return err
	}

	if err := rc.replaceReferencesInData(); err != nil {
		return err
	}

	return nil
}

func (rc *RootConfig) escapeReferences() error {
	zerologr.V(100).Info("Escaping references")
	rc.escapedData = rc.data
	i := 0
	for i < len(rc.data) {
		//nolint:gocritic // ignore: nestedIfs
		if rc.data[i] == '$' && isReference(rc.data[i:i+6]) && rc.data[i-1] != '"' {
			zerologr.V(100).Info(
				fmt.Sprintf("Found unescaped reference at index %d", i),
			)

			end := bytes.IndexByte(rc.data[i:], '}')
			if end == -1 {
				return errors.New("malformed reference: missing closing '}'")
			}

			escapedRef := append([]byte{'"'}, rc.data[i:i+end+1]...)
			escapedRef = append(escapedRef, '"')

			zerologr.V(100).Info("Escaped reference: " + string(escapedRef))

			rc.escapedData = bytes.Replace(rc.escapedData, rc.data[i:i+end+1], escapedRef, 1)

			zerologr.V(100).Info("Intermediate escaped data: \n" + string(rc.escapedData))

			i = i + end + 3
		} else if rc.data[i] == '$' && isReference(rc.data[i:i+6]) {
			zerologr.V(100).Info(
				fmt.Sprintf("Found already escaped reference at index %d", i),
			)

			end := bytes.IndexByte(rc.data[i:], '}')
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

func (rc *RootConfig) walkForReferences() error {
	if len(rc.data) == 0 {
		return nil
	}

	zerologr.V(100).Info("Gathering references")

	generic := make(map[string]any)
	if err := json.Unmarshal(rc.escapedData, &generic); err != nil {
		return err
	}

	if err := rc.walk("", generic); err != nil {
		return err
	}

	zerologr.V(100).Info("Gathered references")
	zerologr.V(100).Info(fmt.Sprintf("Current values map: %+v", rc.values))
	zerologr.V(100).Info(fmt.Sprintf("Current refs map: %+v", rc.refs))

	return nil
}

func (rc *RootConfig) walk(currentPath string, generic any) error {
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

			if err := rc.walk(newPath, v); err != nil {
				return err
			}
		}
	case []any:
		zerologr.V(100).Info("Walking into array in path '" + currentPath + "'")

		for i, item := range val {
			if err := rc.walk(currentPath+"["+strconv.Itoa(i)+"]", item); err != nil {
				return err
			}
		}
	case string:
		if isReference([]byte(val)) {
			zerologr.V(100).Info("Found ref: " + val)
			rc.refs[val] = ""
		}
		rc.values[currentPath] = val
	default:
		zerologr.V(100).Info("Storing final value for path '" + currentPath + "'")
		rc.values[currentPath] = val
	}

	return nil
}

func (rc *RootConfig) findReferenceValues() error {
	zerologr.V(100).Info("Finding reference values")
	/*
		Values contain values for full paths, but some contains references as well. Now what's needed is:

		For values for a given path: return the value if it's not a reference.
		For references: find if the reference can be walked to a value. Environment variables can be resolved
		immediately, path references need to be walked through the values map until a final value is found.

		Environment variable references are resolved first, then path references.
	*/
	for ref := range rc.refs {
		var err error
		if isEnvReference(ref) {
			zerologr.V(100).Info("Resolving environment reference value for: " + ref)
			rc.refs[ref], err = getEnvReferenceValue(ref)
			if err != nil {
				return err
			}
		}
	}

	zerologr.V(100).Info(fmt.Sprintf("Realised env refs: %s", rc.refs))

	for ref := range rc.refs {
		var err error
		if isPathReference(ref) {
			zerologr.V(100).Info("Resolving path reference value for: " + ref)

			// Find if the path reference can be walked to a final value
			rc.refs[ref], err = rc.findReferenceValue(ref)
			if err != nil {
				return err
			}

			zerologr.V(100).
				Info("Resolved path reference value for: " + ref + ", value: " + rc.refs[ref])
		}
	}

	zerologr.V(100).Info(fmt.Sprintf("Realised all refs: %s", rc.refs))

	return nil
}

func (rc *RootConfig) findReferenceValue(ref string) (string, error) {
	zerologr.V(100).Info("Finding value for path reference: " + ref)

	valuePath, err := getPathFromReference(ref)
	if err != nil {
		return "", err
	}

	value, ok := rc.values[valuePath]
	if !ok {
		return "", errors.New("referenced path was not found: " + valuePath)
	}

	decoded, ok := value.(string)
	if ok {
		if isEnvReference(decoded) {
			zerologr.V(100).Info("Nested environment reference found: " + decoded)
			return rc.refs[decoded], nil
		} else if isPathReference(decoded) {
			zerologr.V(100).Info("Nested path reference found, walking path: " + decoded)
			return rc.walkRefs(valuePath, decoded)
		}
	}

	return fmt.Sprint(value), nil
}

func (rc *RootConfig) walkRefs(originPath, ref string) (string, error) {
	zerologr.V(100).Info("Walking references, origin: " + originPath + ", ref: " + ref)

	valuePath, err := getPathFromReference(ref)
	if err != nil {
		return "", err
	}

	if originPath == valuePath {
		return "", errors.New("circular reference detected: " + originPath)
	}

	value, ok := rc.values[valuePath]
	if !ok {
		return "", errors.New("reference path not found: " + valuePath)
	}

	decoded, ok := value.(string)
	if ok {
		if isEnvReference(decoded) {
			zerologr.V(100).Info("Env ref found during walk: " + decoded)
			return rc.refs[decoded], nil
		} else if isPathReference(decoded) {
			zerologr.V(100).Info("Path ref found during walk: " + decoded)
			// Don't decode the reference path here, it's done recursively in the next call to walkRefs.
			// The next call to walkRefs will extract the path from the reference and compare it to the originPath.
			return rc.walkRefs(originPath, decoded)
		}
	}
	zerologr.V(100).
		Info("Final value found for " + originPath + " during walk: " + fmt.Sprint(value))

	return fmt.Sprint(value), nil
}

func (rc *RootConfig) replaceReferencesInData() error {
	/*
		Replace all references in the original JSON data with their resolved values.
	*/
	if len(rc.data) == 0 {
		return nil
	}

	dataStr := string(rc.data)
	zerologr.V(100).Info("Replacing references: " + dataStr)

	for ref, val := range rc.refs {
		zerologr.V(100).Info(
			fmt.Sprintf("Replacing reference '%s' with value '%s'", ref, val),
		)
		dataStr = strings.ReplaceAll(dataStr, ref, val)
		zerologr.V(100).Info("Intermediate replaced data: " + dataStr)
	}

	rc.data = []byte(dataStr)

	zerologr.V(100).Info("Replaced all references: " + string(rc.data))

	return nil
}

func (rc *RootConfig) validateSchema() error {
	zerologr.V(100).Info("Validating schema")

	if len(rc.data) == 0 {
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

	result, err := compiledSchema.Validate(gojsonschema.NewBytesLoader(rc.data))
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

func (rc *RootConfig) loadData() error {
	return json.Unmarshal(rc.data, rc)
}

// postProcess performs post-processing on the loaded configuration, such as setting default values for missing fields,
// in accordance with the configuration schema.
func (rc *RootConfig) postProcess() {
	if rc.AuthConfig != nil {
		rc.AuthConfig.postProcess()
	}
	if rc.OASConfig != nil {
		rc.OASConfig.postProcess()
	}
	rc.RouterConfig.postProcess()
	rc.ObservabilityConfig.postProcess()
	rc.AdminConfig.postProcess()
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
