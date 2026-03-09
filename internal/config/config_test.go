package config

import (
	"os"
	"testing"
)

func TestConfigBad(t *testing.T) {
	// zerologr.Set(zerologr.New(&zerologr.Opts{V: 100, Console: true}))
	// defer func() {
	// 	zerologr.Set(zerologr.New(&zerologr.Opts{V: 0}))
	// }()

	data, err := os.ReadFile("./testconfig/bad.json")
	if err != nil {
		t.Fatalf("failed to read test config: %v", err)
	}

	cfg := New()
	cfg.Load(data)
	if err := cfg.Parse(); err == nil {
		t.Fatalf("expected error when loading bad config, got nil")
	}
}

func TestConfigReferences(t *testing.T) {
	// zerologr.Set(zerologr.New(&zerologr.Opts{V: 100, Console: true}))
	// defer func() {
	// 	zerologr.Set(zerologr.New(&zerologr.Opts{V: 0}))
	// }()

	data, err := os.ReadFile("./testconfig/testconfig.json")
	if err != nil {
		t.Fatalf("failed to read test config: %v", err)
	}

	cfg := New()
	cfg.Load(data)
	if err := cfg.Parse(); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
}

func TestConfigAdminDefaults(t *testing.T) {
	data, err := os.ReadFile("./testconfig/testconfig.json")
	if err != nil {
		t.Fatalf("failed to read test config: %v", err)
	}

	cfg := New()
	cfg.Load(data)
	if err := cfg.Parse(); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.AdminConfig.SuperUser.ClientID != "admin" {
		t.Errorf("expected superuser client ID to be 'admin', got '%s'", cfg.AdminConfig.SuperUser.ClientID)
	}

	if cfg.AdminConfig.SuperUser.ClientSecret != "secret" {
		t.Errorf("expected superuser client secret to be 'secret', got '%s'", cfg.AdminConfig.SuperUser.ClientSecret)
	}
}

func TestConfigNoRouter(t *testing.T) {
	data, err := os.ReadFile("./testconfig/testconfig_norouter.json")
	if err != nil {
		t.Fatalf("failed to read test config: %v", err)
	}

	cfg := New()
	cfg.Load(data)
	if err := cfg.Parse(); err == nil {
		t.Fatalf("expected error when loading config without router, got nil")
	}
}

func TestConfigEmpty(t *testing.T) {
	data, err := os.ReadFile("./testconfig/empty.json")
	if err != nil {
		t.Fatalf("failed to read test config: %v", err)
	}

	cfg := New()
	cfg.Load(data)
	if err := cfg.Parse(); err == nil {
		t.Fatalf("expected error when loading empty config, got nil")
	}
}

func TestConfigOAS(t *testing.T) {
	data, err := os.ReadFile("./testconfig/testconfig_oas.json")
	if err != nil {
		t.Fatalf("failed to read test config: %v", err)
	}

	cfg := New()
	cfg.Load(data)
	if err := cfg.Parse(); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if len(cfg.OASConfig.Mappings) != 1 {
		t.Fatalf("expected 1 OAS mapping, got %d", len(cfg.OASConfig.Mappings))
	}

	mapping := cfg.OASConfig.Mappings[0]
	if mapping.Backend != "backend1" {
		t.Errorf("expected OAS mapping backend to be 'backend1', got '%s'", mapping.Backend)
	}

	if mapping.Specification != "./testconfig/oas_spec.yaml" {
		t.Errorf("expected OAS mapping specification to be './testconfig/oas_spec.yaml', got '%s'", mapping.Specification)
	}

	if mapping.Options == nil {
		t.Fatalf("expected OAS mapping options to be non-nil, got nil")
	}

	if !mapping.Options.ValidateBody {
		t.Errorf("expected OAS mapping options to have ValidateBody=true, got false")
	}
}
