package auth

import (
	"os"
	"testing"

	"github.com/trebent/kerberos/internal/composer/custom"
	"github.com/trebent/kerberos/internal/config"
	"github.com/xeipuuv/gojsonschema"
)

func TestConfigMissingMethod(t *testing.T) {
	ac := &authConfig{}

	configData, err := os.ReadFile("./testconfigs/missing-methods.json")
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	sl := gojsonschema.NewSchemaLoader()
	sl.AddSchemas(custom.OrderedSchemaJSONLoader())
	schema, err := sl.Compile(ac.SchemaJSONLoader())
	if err != nil {
		t.Fatalf("Failed to compile schema: %v", err)
	}

	result, err := schema.Validate(gojsonschema.NewBytesLoader(configData))
	if err != nil {
		t.Fatalf("Got error: %v", err)
	}

	if len(result.Errors()) == 0 {
		t.Fatal("Got no errors")
	}

	for _, e := range result.Errors() {
		t.Log(e.Description())
	}
}

func TestConfigMissingScheme(t *testing.T) {
	ac := &authConfig{}

	configData, err := os.ReadFile("./testconfigs/missing-scheme.json")
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	sl := gojsonschema.NewSchemaLoader()
	sl.AddSchemas(custom.OrderedSchemaJSONLoader())
	schema, err := sl.Compile(ac.SchemaJSONLoader())
	if err != nil {
		t.Fatalf("Failed to compile schema: %v", err)
	}

	result, err := schema.Validate(gojsonschema.NewBytesLoader(configData))
	if err != nil {
		t.Fatalf("Got error: %v", err)
	}

	if len(result.Errors()) == 0 {
		t.Fatal("Got no errors")
	}

	for _, e := range result.Errors() {
		t.Log(e.Description())
	}
}

func TestConfigMissingSchemeMappings(t *testing.T) {
	ac := &authConfig{}

	configData, err := os.ReadFile("./testconfigs/missing-scheme-mappings.json")
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	sl := gojsonschema.NewSchemaLoader()
	sl.AddSchemas(custom.OrderedSchemaJSONLoader())
	schema, err := sl.Compile(ac.SchemaJSONLoader())
	if err != nil {
		t.Fatalf("Failed to compile schema: %v", err)
	}

	result, err := schema.Validate(gojsonschema.NewBytesLoader(configData))
	if err != nil {
		t.Fatalf("Got error: %v", err)
	}

	if len(result.Errors()) == 0 {
		t.Fatal("Got no errors")
	}

	for _, e := range result.Errors() {
		t.Log(e.Description())
	}
}

func TestConfigOnlyAdmin(t *testing.T) {
	ac := &authConfig{}

	configData, err := os.ReadFile("./testconfigs/only-admin.json")
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	sl := gojsonschema.NewSchemaLoader()
	sl.AddSchemas(custom.OrderedSchemaJSONLoader())
	schema, err := sl.Compile(ac.SchemaJSONLoader())
	if err != nil {
		t.Fatalf("Failed to compile schema: %v", err)
	}

	result, err := schema.Validate(gojsonschema.NewBytesLoader(configData))
	if err != nil {
		t.Fatalf("Got error: %v", err)
	}

	if len(result.Errors()) != 0 {
		for _, e := range result.Errors() {
			t.Error(e.Description())
		}

		t.Fatal("Got errors")
	}
}

func TestConfigAdminSuperUserDefault(t *testing.T) {
	cm := config.New(&config.Opts{})
	_, err := RegisterWith(cm)
	if err != nil {
		t.Fatalf("Failed to register auth config: %v", err)
	}

	ac := config.AccessAs[*authConfig](cm, configName)
	if ac.Administration.SuperUser.ClientID != defaultSuperUserClientID {
		t.Fatal("Client ID was not defaulted")
	}

	if ac.Administration.SuperUser.ClientSecret != defaultSuperUserClientSecret {
		t.Fatal("Client secret was not defaulted")
	}
}

func TestConfigExempt(t *testing.T) {
	ac := &authConfig{}

	configData, err := os.ReadFile("./testconfigs/exempt.json")
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	sl := gojsonschema.NewSchemaLoader()
	sl.AddSchemas(custom.OrderedSchemaJSONLoader())
	schema, err := sl.Compile(ac.SchemaJSONLoader())
	if err != nil {
		t.Fatalf("Failed to compile schema: %v", err)
	}

	result, err := schema.Validate(gojsonschema.NewBytesLoader(configData))
	if err != nil {
		t.Fatalf("Got error: %v", err)
	}

	if len(result.Errors()) != 0 {
		for _, e := range result.Errors() {
			t.Error(e.Description())
		}

		t.Fatal("Got errors")
	}
}

func TestConfigAuthorization(t *testing.T) {
	ac := &authConfig{}

	configData, err := os.ReadFile("./testconfigs/authorization.json")
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	sl := gojsonschema.NewSchemaLoader()
	sl.AddSchemas(custom.OrderedSchemaJSONLoader())
	schema, err := sl.Compile(ac.SchemaJSONLoader())
	if err != nil {
		t.Fatalf("Failed to compile schema: %v", err)
	}

	result, err := schema.Validate(gojsonschema.NewBytesLoader(configData))
	if err != nil {
		t.Fatalf("Got error: %v", err)
	}

	if len(result.Errors()) != 0 {
		for _, e := range result.Errors() {
			t.Error(e.Description())
		}

		t.Fatal("Got errors")
	}
}
