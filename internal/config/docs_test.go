package config

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/trebent/kerberos/internal/docscheck"
	"github.com/trebent/schemer"
)

// The docs keep one page per config section, so that a key, its type and its default are written
// down in exactly one place. These tests fail when that page drifts from the schemas and the code.
// docs/configuration.md describes the layout; see "Root keys" there.

// docOwners maps a config path to the page that documents it. A path not listed here is owned by
// the page of its closest listed parent.
//
//nolint:gochecknoglobals // test fixture
var docOwners = map[string]string{
	"":                              "docs/configuration.md",
	"gateway":                       "docs/routing.md",
	"gateway.tls":                   "docs/tls.md",
	"gateway.router.backends[].tls": "docs/tls.md",
	"admin":                         "docs/admin-api.md",
	"admin.api.tls":                 "docs/tls.md",
	"auth":                          "docs/authentication.md",
	"oas":                           "docs/request-validation.md",
	"observability":                 "docs/observability.md",
	"persistence":                   "docs/persistence.md",
}

// sharedOwners maps the schemas that are reused in several places to the page that documents them
// everywhere they appear.
//
//nolint:gochecknoglobals // test fixture
var sharedOwners = map[string]string{
	"cookies_schema.json":     "docs/sessions.md",
	"origins_schema.json":     "docs/cors.md",
	"ordered_schema.json":     "docs/routing.md",
	"persistence_schema.json": "docs/persistence.md",
}

const (
	minimalGateway     = `"gateway": {"router": {"backends": [{"name": "echo", "host": "echo", "port": 15000}]}}`
	minimalPersistence = `"persistence": {"driver": "sqlite", "address": "krb.db"}`
)

var (
	unquotedEnvRefRe = regexp.MustCompile(`([:\[,]\s*)\$\{env:(\w+)\}`)
	quotedEnvRefRe   = regexp.MustCompile(`"\$\{env:(\w+)\}"`)
	anyRefRe         = regexp.MustCompile(`\$\{(env|ref):[^}]*\}`)
)

func loadSchemaFiles(t *testing.T) map[string]map[string]any {
	t.Helper()

	files, err := filepath.Glob("schemas/*.json")
	if err != nil {
		t.Fatal(err)
	}

	schemas := map[string]map[string]any{}

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}

		var s map[string]any
		if err := json.Unmarshal(data, &s); err != nil {
			t.Fatalf("%s: %v", f, err)
		}

		schemas[filepath.Base(f)] = s
	}

	return schemas
}

// TestDocsRootKeys checks that docs/configuration.md lists every top-level key and links each one
// to the page that owns it.
func TestDocsRootKeys(t *testing.T) {
	md := docscheck.ReadDoc(t, "docs/configuration.md")
	props, _ := loadSchemaFiles(t)["config_schema.json"]["properties"].(map[string]any)

	for _, key := range slices.Sorted(maps.Keys(props)) {
		cell, ok := docscheck.TableCell(md, "`"+key+"`", "Documented in")
		if !ok {
			t.Errorf("docs/configuration.md: root key %q has no row in the root keys table", key)
			continue
		}

		if owner := path.Base(docOwners[key]); !strings.Contains(cell, "(./"+owner) {
			t.Errorf(
				"docs/configuration.md: root key %q should link to %s, got %q",
				key,
				owner,
				cell,
			)
		}
	}
}

// TestDocsSchemaKeys checks that every key in the schemas is mentioned by the page that owns it.
func TestDocsSchemaKeys(t *testing.T) {
	schemas := loadSchemaFiles(t)
	docs := map[string]string{}

	read := func(doc string) string {
		if _, ok := docs[doc]; !ok {
			docs[doc] = docscheck.ReadDoc(t, doc)
		}

		return docs[doc]
	}

	var walk func(node map[string]any, at, owner string)
	walk = func(node map[string]any, at, owner string) {
		if ref, ok := node["$ref"].(string); ok {
			file := path.Base(ref)
			if shared, ok := sharedOwners[file]; ok {
				owner = shared
				if !strings.Contains(read(owner), file) {
					t.Errorf("%s: should link to the schema %s that it documents", owner, file)
				}
			}

			walk(schemas[file], at, owner)

			return
		}

		if items, ok := node["items"].(map[string]any); ok {
			walk(items, at+"[]", owner)
		}

		props, _ := node["properties"].(map[string]any)
		for _, key := range slices.Sorted(maps.Keys(props)) {
			// auth_schema.json and cookies_schema.json list "additionalProperties" as a property
			// by mistake; it is not a config key.
			if key == "additionalProperties" {
				continue
			}

			childAt := strings.TrimPrefix(at+"."+key, ".")
			childOwner := owner

			if o, ok := docOwners[childAt]; ok {
				childOwner = o
			}

			if !docscheck.MentionsKey(read(childOwner), key) {
				t.Errorf("%s: does not document the config key %s", childOwner, childAt)
			}

			child, _ := props[key].(map[string]any)
			walk(child, childAt, childOwner)
		}
	}

	walk(schemas["config_schema.json"], "", docOwners[""])
	walk(schemas["connector_schema.json"], "", "docs/admin-connector.md")
}

// TestDocsDefaults checks the defaults stated by the owning pages against the code.
func TestDocsDefaults(t *testing.T) {
	cfg := parseDocExample(t, NewKerberosSchemer(), NewKerberos(), `{
		`+minimalGateway+`,
		"auth": {
			"methods": {"basic": {}},
			"scheme": {"mappings": [{"backend": "echo", "method": "basic"}]}
		},
		"oas": {"mappings": [{"backend": "echo", "specification": "echo.yaml"}]}
	}`)
	if t.Failed() {
		return
	}

	kc, _ := cfg.(*Kerberos)

	tests := []struct {
		doc, row, want string
	}{
		{"docs/routing.md", "`timeout`", strconv.Itoa(kc.GatewayCfg.Router.Backends[0].TimeoutMs)},
		{"docs/routing.md", "`order`", strconv.Itoa(kc.AuthCfg.Order)},
		{"docs/admin-api.md", "`admin.superUser.clientId`", kc.AdminCfg.SuperUser.ClientID},
		{"docs/admin-api.md", "`admin.superUser.clientSecret`", kc.AdminCfg.SuperUser.ClientSecret},
		{"docs/sessions.md", "`sameSite`", kc.AdminCfg.API.Cookies.SameSite},
		{"docs/sessions.md", "`sameSite`", kc.AuthCfg.Methods.Basic.API.Cookies.SameSite},
		{"docs/observability.md", "`enabled`", strconv.FormatBool(kc.ObsCfg.Enabled)},
		{"docs/observability.md", "`runtimeMetrics`", strconv.FormatBool(kc.ObsCfg.RuntimeMetrics)},
		{"docs/persistence.md", "`driver`", kc.PersistenceCfg.Driver},
		{"docs/persistence.md", "`address`", kc.PersistenceCfg.Address},
		{
			"docs/request-validation.md", "`oas.mappings[].options.validateBody`",
			strconv.FormatBool(kc.OASCfg.Mappings[0].Options.ValidateBody),
		},
	}

	for _, tt := range tests {
		md := docscheck.ReadDoc(t, tt.doc)

		got, ok := docscheck.TableCell(md, tt.row, "Default")
		if !ok {
			t.Errorf("%s: no table row %s with a Default column", tt.doc, tt.row)
			continue
		}

		if !strings.HasPrefix(got, "`"+tt.want+"`") {
			t.Errorf(
				"%s: default of %s is documented as %q, the code says `%s`",
				tt.doc,
				tt.row,
				got,
				tt.want,
			)
		}
	}
}

// TestDocsExamples checks that every JSON config example in the docs passes the real config loader.
// Examples that show only some sections are completed with a minimal version of the required one.
func TestDocsExamples(t *testing.T) {
	root, err := docscheck.RepoRoot()
	if err != nil {
		t.Fatal(err)
	}

	kerberosKeys := schemaKeys(t, "config_schema.json")
	connectorKeys := schemaKeys(t, "connector_schema.json")

	err = filepath.WalkDir(
		filepath.Join(root, "docs"),
		func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || filepath.Ext(p) != ".md" {
				return err
			}

			rel, _ := filepath.Rel(root, p)
			isConnectorDoc := filepath.Base(p) == "admin-connector.md"

			for i, block := range docscheck.JSONBlocks(docscheck.ReadDoc(t, rel)) {
				keys := topLevelKeys(block)

				t.Run(fmt.Sprintf("%s#%d", rel, i+1), func(t *testing.T) {
					switch {
					case isConnectorDoc && len(keys) > 0 && isSubset(keys, connectorKeys):
						block = withSection(block, keys, "persistence", minimalPersistence)
						parseDocExample(t, NewConnectorSchemer(), NewConnector(), block)
					case len(keys) > 0 && isSubset(keys, kerberosKeys):
						block = withSection(block, keys, "gateway", minimalGateway)
						parseDocExample(t, NewKerberosSchemer(), NewKerberos(), block)
					default:
						t.Skip("not a config example")
					}
				})
			}

			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
}

// parseDocExample runs an example through the config loader. Environment references without a
// default are given placeholder values so the example can resolve.
func parseDocExample(t *testing.T, sch *schemer.Schemer, target any, example string) any {
	t.Helper()

	for _, m := range unquotedEnvRefRe.FindAllStringSubmatch(example, -1) {
		t.Setenv(m[2], "1")
	}

	for _, m := range quotedEnvRefRe.FindAllStringSubmatch(example, -1) {
		t.Setenv(m[1], "placeholder")
	}

	sch.Load([]byte(example))

	if err := sch.Parse(target); err != nil {
		t.Errorf("example does not load: %v\n%s", err, example)
		return target
	}

	if pp, ok := target.(interface{ PostProcess() }); ok {
		pp.PostProcess()
	}

	return target
}

func schemaKeys(t *testing.T, file string) []string {
	t.Helper()

	props, _ := loadSchemaFiles(t)[file]["properties"].(map[string]any)

	return slices.Collect(maps.Keys(props))
}

// topLevelKeys returns the keys of a JSON object, or nil if block is not one. References are
// replaced first, since unquoted ones are not valid JSON.
func topLevelKeys(block string) []string {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(anyRefRe.ReplaceAllString(block, "0")), &obj); err != nil {
		return nil
	}

	return slices.Collect(maps.Keys(obj))
}

// withSection adds a required section to an example that leaves it out.
func withSection(block string, keys []string, key, section string) string {
	if slices.Contains(keys, key) {
		return block
	}

	return strings.Replace(block, "{", "{"+section+",", 1)
}

func isSubset(keys, of []string) bool {
	for _, k := range keys {
		if !slices.Contains(of, k) {
			return false
		}
	}

	return true
}
