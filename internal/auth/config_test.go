package auth

import (
	"os"
	"testing"

	"github.com/trebent/kerberos/internal/config"
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

func TestConfigOnlyAdmin(t *testing.T) {
	ac := &authConfig{}

	configData, err := os.ReadFile("./testconfigs/only-admin.json")
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	schema := ac.Schema()
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
	cm := config.New()
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
