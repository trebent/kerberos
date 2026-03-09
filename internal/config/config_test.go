package config

import (
	"os"
	"testing"
)

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
