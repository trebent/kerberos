package config

import (
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/kaptinlin/jsonschema"
	"github.com/trebent/zerologr"
)

type (
	testCfg struct {
		Enabled      bool      `json:"enabled" jsonschema:"required"`
		Number       int       `json:"number"`
		String       string    `json:"string"`
		Array        []string  `json:"array"`
		Complex      *subCfg   `json:"complex"`
		ComplexArray []*subCfg `json:"complex_array"`
	}
	subCfg struct {
		Bool   bool   `json:"bool"`
		Number int    `json:"number"`
		String string `json:"string"`
	}
)

func (t *testCfg) Schema() *jsonschema.Schema {
	return jsonschema.FromStruct[testCfg]()
}

func TestLoadNoName(t *testing.T) {
	m := New()

	m.Register("ok", &testCfg{})

	if err := m.Load("ok", []byte{}); err != nil {
	}

	if err := m.Load("nok", []byte{}); err == nil {
		t.Fatal("Should have errored out due to no registered config name")
	}
}

func TestParseEnvRef(t *testing.T) {
	os.Setenv("ENABLED", "true")
	teardown := enableLogging()
	defer teardown()
	defer os.Unsetenv("ENABLED")

	cfg := &testCfg{}
	cfgData := []byte(`{
  "enabled": ${env:ENABLED},
  "complex": {
    "string": "complex.string"
  }
}`)

	m := New()
	m.Register("1", cfg)

	if err := m.Load("1", cfgData); err != nil {
		t.Fatal("Unexpected error when loading config 1:", err)
	}

	if err := m.Parse(); err != nil {
		t.Fatal("Unexpected error when parsing loaded config:", err)
	}

	accessCfg, _ := m.Access("1")
	decodedAccessCfg := accessCfg.(*testCfg)
	if decodedAccessCfg.Complex.String != "complex.string" {
		t.Fatal("Expected comlex.string to contain \"complex.string\"")
	}

	if !decodedAccessCfg.Enabled {
		t.Fatal("Expected enabled to be true")
	}
}

func TestParseEnvRefFailed(t *testing.T) {
	os.Setenv("ENABLED", "true")
	teardown := enableLogging()
	defer teardown()
	defer os.Unsetenv("ENABLED")

	cfg := &testCfg{}
	cfgData := []byte(`{
  "enabled": ${env:ENABLED},
  "complex": {
    "string": "${env:MISSING_ENV_VAR}"
  }
}`)

	m := New()
	m.Register("1", cfg)

	if err := m.Load("1", cfgData); err != nil {
		t.Fatal("Unexpected error when loading config 1:", err)
	}

	err := m.Parse()
	if err == nil {
		t.Fatal("Expected error when parsing loaded config")
	}

	t.Log("Received expected error when parsing loaded config:", err)
}

func TestParseEnvRefDefault(t *testing.T) {
	os.Setenv("ENABLED", "true")
	teardown := enableLogging()
	defer teardown()
	defer os.Unsetenv("ENABLED")

	cfg := &testCfg{}
	cfgData := []byte(`{
  "enabled": ${env:ENABLED},
  "complex": {
    "string": "${env:MISSING_ENV_VAR:default_value}"
  }
}`)

	m := New()
	m.Register("1", cfg)

	if err := m.Load("1", cfgData); err != nil {
		t.Fatal("Unexpected error when loading config 1:", err)
	}

	err := m.Parse()
	if err != nil {
		t.Fatal("Unexpected error when parsing loaded config:", err)
	}

	accessCfg, _ := m.Access("1")
	decodedAccessCfg := accessCfg.(*testCfg)
	if decodedAccessCfg.Complex.String != "default_value" {
		t.Fatal("Expected comlex.string to contain \"default_value\"")
	}

	if !decodedAccessCfg.Enabled {
		t.Fatal("Expected enabled to be true")
	}
}

func TestParsePathRef(t *testing.T) {
	teardown := enableLogging()
	defer teardown()
	os.Setenv("STRING_VALUE", "top.string")
	defer os.Unsetenv("STRING_VALUE")

	cfg := &testCfg{}

	data := []byte(`{
  "enabled": true,
  "string": "${env:STRING_VALUE}",
  "complex": {
		"bool": ${ref:1.enabled},
		"string": "${ref:1.string}"
  }
}`)

	m := New()
	m.Register("1", cfg)

	if err := m.Load("1", data); err != nil {
		t.Fatal("Unexpected error when loading config 1:", err)
	}

	err := m.Parse()
	if err != nil {
		t.Fatal("Unexpected error when parsing loaded config:", err)
	}

	accessCfg, _ := m.Access("1")
	decodedAccessCfg := accessCfg.(*testCfg)

	if decodedAccessCfg.Complex.String != "top.string" {
		t.Fatal("Expected complex.string to contain \"top.string\"")
	}
}

func enableLogging() func() {
	newLogger := zerologr.New(&zerologr.Opts{Console: true, V: 100})
	zerologr.Set(newLogger)
	return func() {
		zerologr.Set(logr.Logger{})
	}
}
