package auth

import (
	"os"
	"testing"

	"github.com/xeipuuv/gojsonschema"
)

func TestConfigMissingMethod(t *testing.T) {
	ac := &authConfig{}

	configData, err := os.ReadFile("./testconfigs/missing-methods.json")
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	schema := ac.Schema()
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

	schema := ac.Schema()
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

	schema := ac.Schema()
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
