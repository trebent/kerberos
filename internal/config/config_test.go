package config

import (
	"errors"
	"os"
	"strconv"
	"testing"

	_ "embed"

	"github.com/go-logr/logr"
	"github.com/trebent/zerologr"
	"github.com/xeipuuv/gojsonschema"
)

type (
	testCfg struct {
		Enabled      bool      `json:"enabled"`
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
	globalCfg struct {
		Identifier string `json:"identifier"`
		Enabled    bool   `json:"enabled"`
	}
	derivedCfg struct {
		testCfg   `json:"test"`
		globalCfg `json:"global"`
	}
	noSchema struct {
		Bool   bool   `json:"bool"`
		Number int    `json:"number"`
		String string `json:"string"`
	}
)

var (
	//go:embed testschemas/testcfg_schema.json
	testCfgSchema []byte
	//go:embed testschemas/global_schema.json
	globalCfgSchema []byte
	//go:embed testschemas/derived_schema.json
	derivedCfgSchema []byte
)

func (t *testCfg) Schema() *gojsonschema.Schema {
	s, err := gojsonschema.NewSchema(t.SchemaJSONLoader())
	if err != nil {
		panic("Failed to create schema for testCfg: " + err.Error())
	}

	return s
}

func (t *testCfg) SchemaJSONLoader() gojsonschema.JSONLoader {
	return gojsonschema.NewBytesLoader(testCfgSchema)
}

func (n *derivedCfg) SchemaJSONLoader() gojsonschema.JSONLoader {
	return gojsonschema.NewBytesLoader(derivedCfgSchema)
}

func (n *globalCfg) SchemaJSONLoader() gojsonschema.JSONLoader {
	return gojsonschema.NewBytesLoader(globalCfgSchema)
}

func (n *noSchema) SchemaJSONLoader() gojsonschema.JSONLoader {
	return nil
}

func TestMultipleDerivedConfigs(t *testing.T) {
	disable := enableLogging()
	defer disable()

	dc := &derivedCfg{}
	dc2 := &derivedCfg{}
	tc := &testCfg{}
	gc := &globalCfg{}
	m := New(&Opts{GlobalSchemas: []gojsonschema.JSONLoader{
		gc.SchemaJSONLoader(),
		tc.SchemaJSONLoader(),
	}})

	m.Register("derived", dc)
	m.Register("derived2", dc2)
	if err := m.Load("derived", []byte(`{
	"global": {
		"enabled": true,
		"identifier": "set_in_global"
	}, 
	"test": {
		"enabled": true
	}
}`,
	)); err != nil {
		t.Fatalf("Unexpected error when loading registered config name: %v", err)
	}

	if err := m.Load("derived2", []byte(`{
	"global": {
		"enabled": true,
		"identifier": "set_in_global"
	}, 
	"test": {
		"enabled": true
	}
}`,
	)); err != nil {
		t.Fatalf("Unexpected error when loading registered config name: %v", err)
	}

	if err := m.Parse(); err != nil {
		t.Fatalf("Unexpected error when parsing loaded config: %v", err)
	}

	accessCfg, _ := m.Access("derived")
	decodedAccessCfg := accessCfg.(*derivedCfg)

	if !decodedAccessCfg.globalCfg.Enabled {
		t.Fatalf("Expected globalCfg.Enabled to be true, got: %v", decodedAccessCfg.globalCfg.Enabled)
	}
	if decodedAccessCfg.globalCfg.Identifier != "set_in_global" {
		t.Fatalf("Expected globalCfg.Identifier to be \"set_in_global\", got: %s", decodedAccessCfg.globalCfg.Identifier)
	}
	if !decodedAccessCfg.testCfg.Enabled {
		t.Fatalf("Expected testCfg.Enabled to be true, got: %v", decodedAccessCfg.testCfg.Enabled)
	}
}

func TestDerivedMissingRef(t *testing.T) {
	disable := enableLogging()
	defer disable()

	dc := &derivedCfg{}
	// tc := &testCfg{}
	gc := &globalCfg{}
	m := New(&Opts{GlobalSchemas: []gojsonschema.JSONLoader{
		gc.SchemaJSONLoader(),
		// Makes validation fail due to missing test reference.
		// tc.SchemaJSONLoader(),
	}})

	m.Register("bad_derived", dc)
	if err := m.Load("bad_derived", []byte(`{}`)); err != nil {
		t.Fatalf("Unexpected error when loading registered config name: %v", err)
	}

	if err := m.Parse(); err == nil {
		t.Fatal("Expected error when parsing loaded config")
	} else {
		t.Logf("Received expected error when parsing loaded config: %v", err)
	}
}

func TestDerivedConfig(t *testing.T) {
	disable := enableLogging()
	defer disable()

	dc := &derivedCfg{}
	tc := &testCfg{}
	gc := &globalCfg{}
	m := New(&Opts{GlobalSchemas: []gojsonschema.JSONLoader{
		gc.SchemaJSONLoader(),
		tc.SchemaJSONLoader(),
	}})

	m.Register("derived", dc)
	if err := m.Load("derived", []byte(`{
	"global": {
		"enabled": true,
		"identifier": "set_in_global"
	}, 
	"test": {
		"enabled": true
	}
}`,
	)); err != nil {
		t.Fatalf("Unexpected error when loading registered config name: %v", err)
	}

	if err := m.Parse(); err != nil {
		t.Fatalf("Unexpected error when parsing loaded config: %v", err)
	}

	accessCfg, _ := m.Access("derived")
	decodedAccessCfg := accessCfg.(*derivedCfg)

	if !decodedAccessCfg.globalCfg.Enabled {
		t.Fatalf("Expected globalCfg.Enabled to be true, got: %v", decodedAccessCfg.globalCfg.Enabled)
	}
	if decodedAccessCfg.globalCfg.Identifier != "set_in_global" {
		t.Fatalf("Expected globalCfg.Identifier to be \"set_in_global\", got: %s", decodedAccessCfg.globalCfg.Identifier)
	}
	if !decodedAccessCfg.testCfg.Enabled {
		t.Fatalf("Expected testCfg.Enabled to be true, got: %v", decodedAccessCfg.testCfg.Enabled)
	}
}

func TestLoadNoName(t *testing.T) {
	m := New(&Opts{})
	m.Register("ok", &testCfg{})

	if err := m.Load("ok", []byte{}); err != nil {
		t.Fatalf("Unexpected error when loading registered config name: %v", err)
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

	m := New(&Opts{})
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

	m := New(&Opts{})
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

	m := New(&Opts{})
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

	m := New(&Opts{})
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

	if decodedAccessCfg.Enabled != true {
		t.Fatal("Expected enabled to be true")
	}

	if decodedAccessCfg.Complex.Bool != true {
		t.Fatal("Expected complex.bool to be true")
	}
}

func TestParsePathRefCircular(t *testing.T) {
	teardown := enableLogging()
	defer teardown()

	cfg := &testCfg{}

	data := []byte(`{
  "enabled": true,
  "string": "top.string",
  "complex": {
		"bool": ${ref:1.enabled},
		"string": "${ref:1.complex.string}"
  }
}`)

	m := New(&Opts{})
	m.Register("1", cfg)

	if err := m.Load("1", data); err != nil {
		t.Fatal("Unexpected error when loading config 1:", err)
	}

	err := m.Parse()
	if !errors.Is(err, ErrPathVarRefCircular) {
		t.Fatal("Unexpected error when parsing loaded config:", err)
	}
}

func TestParsePathRefCircularBackRef(t *testing.T) {
	teardown := enableLogging()
	defer teardown()

	cfg := &testCfg{}

	data := []byte(`{
  "enabled": true,
  "string": "${ref:1.complex.string}",
  "complex": {
		"bool": ${ref:1.enabled},
		"string": "${ref:1.string}"
  }
}`)

	m := New(&Opts{})
	m.Register("1", cfg)

	if err := m.Load("1", data); err != nil {
		t.Fatal("Unexpected error when loading config 1:", err)
	}

	err := m.Parse()
	if !errors.Is(err, ErrPathVarRefCircular) {
		t.Fatal("Unexpected error when parsing loaded config:", err)
	}

	t.Log(err.Error())
}

func TestParsePathRefArrayIndex(t *testing.T) {
	teardown := enableLogging()
	defer teardown()

	cfg := &testCfg{}

	data := []byte(`{
  "enabled": true,
  "string": "${ref:1.complex_array[0].string}",
  "complex": {
		"bool": ${ref:1.enabled},
		"string": "${ref:1.string}"
  },
	"complex_array": [
			{
				"string": "index0"
      }
		]
}`)

	m := New(&Opts{})
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

	if decodedAccessCfg.String != "index0" {
		t.Fatal("Expected string to contain \"index0\"")
	}

	if decodedAccessCfg.Complex.String != "index0" {
		t.Fatal("Expected string to contain \"index0\"")
	}
}

func TestParsePathRefToEnvRef(t *testing.T) {
	teardown := enableLogging()
	defer teardown()
	os.Setenv("STRING_VALUE", "env_string")
	defer os.Unsetenv("STRING_VALUE")

	cfg := &testCfg{}

	data := []byte(`{
  "enabled": true,
  "string": "${ref:1.complex_array[0].string}",
	"complex": {
		"string": "${env:STRING_VALUE}"
	},
	"complex_array": [
			{
				"string": "${ref:1.complex.string}"
      }
		]
}`)

	m := New(&Opts{})
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

	if decodedAccessCfg.String != "env_string" {
		t.Fatal("Expected string to contain \"env_string\"")
	}

	if decodedAccessCfg.ComplexArray[0].String != "env_string" {
		t.Fatal("Expected string to contain \"env_string\"")
	}
}

func TestParsePathRefCrossDocument(t *testing.T) {
	teardown := enableLogging()
	defer teardown()
	os.Setenv("STRING_VALUE", "env_string")
	defer os.Unsetenv("STRING_VALUE")

	cfg := &testCfg{}

	data := []byte(`{
  "enabled": true,
  "string": "1string"
}`)
	data2 := []byte(`{
  "enabled": true,
  "string": "${ref:1.string}"
}`)

	m := New(&Opts{})
	m.Register("1", cfg)
	m.Register("2", cfg)

	if err := m.Load("1", data); err != nil {
		t.Fatal("Unexpected error when loading config 1:", err)
	}
	if err := m.Load("2", data2); err != nil {
		t.Fatal("Unexpected error when loading config 1:", err)
	}

	err := m.Parse()
	if err != nil {
		t.Fatal("Unexpected error when parsing loaded config:", err)
	}

	accessCfg, _ := m.Access("2")
	decodedAccessCfg := accessCfg.(*testCfg)

	if decodedAccessCfg.String != "1string" {
		t.Fatal("Expected string to contain \"1string\"")
	}
}

func TestParseJSONSchemaValidationFail(t *testing.T) {
	teardown := enableLogging()
	defer teardown()

	cfg := &testCfg{}

	// Enabled is required but missing
	data := []byte(`{}`)

	m := New(&Opts{})
	m.Register("1", cfg)

	if err := m.Load("1", data); err != nil {
		t.Fatal("Unexpected error when loading config 1:", err)
	}

	err := m.Parse()
	if err == nil {
		t.Fatal("Expected error when parsing loaded config, got nil")
	} else {
		t.Log(err)
	}
}

func TestParseJSONSchemaValidationExtraField(t *testing.T) {
	teardown := enableLogging()
	defer teardown()

	cfg := &testCfg{}

	data := []byte(`{
	"enabled": true,
	"unknown": "value"	
}`)

	m := New(&Opts{})
	m.Register("1", cfg)

	if err := m.Load("1", data); err != nil {
		t.Fatal("Unexpected error when loading config 1:", err)
	}

	err := m.Parse()
	if err == nil {
		t.Fatal("Expected error when parsing loaded config, got nil")
	} else {
		t.Log(err)
	}
}

func TestParseJSONSchemaValidationWrongType(t *testing.T) {
	teardown := enableLogging()
	defer teardown()

	cfg := &testCfg{}

	// Enabled is the wrong type
	data := []byte(`{
	"enabled": "true"	
}`)

	m := New(&Opts{})
	m.Register("1", cfg)

	if err := m.Load("1", data); err != nil {
		t.Fatal("Unexpected error when loading config 1:", err)
	}

	err := m.Parse()
	if err == nil {
		t.Fatal("Expected error when parsing loaded config, got nil")
	} else {
		t.Log(err)
	}
}

func TestParseNoSchema(t *testing.T) {
	teardown := enableLogging()
	defer teardown()

	cfg := &noSchema{}

	// Enabled is the wrong type
	data := []byte(`{
	"bool": true,
	"number": 42,
	"string": "test"	
}`)

	m := New(&Opts{})
	m.Register("1", cfg)

	if err := m.Load("1", data); err != nil {
		t.Fatal("Unexpected error when loading config 1:", err)
	}

	err := m.Parse()
	if err != nil {
		t.Fatal("Unexpected error when parsing loaded config:", err)
	}
}

func enableLogging() func() {
	v := 0
	val, ok := os.LookupEnv("LOG_VERBOSITY")
	if ok {
		var err error
		v, err = strconv.Atoi(val)
		if err != nil {
			v = 0
		}
	}

	newLogger := zerologr.New(&zerologr.Opts{Console: true, V: v})
	zerologr.Set(newLogger)
	return func() {
		zerologr.Set(logr.Logger{})
	}
}
