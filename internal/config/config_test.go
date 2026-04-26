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

	data, err := os.ReadFile("./testconfig/unknown_field.json")
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
	data, err := os.ReadFile("./testconfig/testconfig_nogw.json")
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

func TestConfigPersistence(t *testing.T) {
	data, err := os.ReadFile("./testconfig/testconfig_persistence.json")
	if err != nil {
		t.Fatalf("failed to read test config: %v", err)
	}

	cfg := New()
	cfg.Load(data)
	if err := cfg.Parse(); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.PersistenceConfig.Driver != "postgres" {
		t.Errorf("expected persistence driver to be 'postgres', got '%s'", cfg.PersistenceConfig.Driver)
	}

	if cfg.PersistenceConfig.Address != "localhost:5432" {
		t.Errorf("expected persistence address to be 'localhost', got '%s'", cfg.PersistenceConfig.Address)
	}

	if cfg.PersistenceConfig.Database != "kerberos" {
		t.Errorf("expected persistence database to be 'kerberos', got '%s'", cfg.PersistenceConfig.Database)
	}

	if *cfg.PersistenceConfig.Username != "user" {
		t.Errorf("expected persistence user to be 'user', got '%s'", *cfg.PersistenceConfig.Postgres.Username)
	}

	if *cfg.PersistenceConfig.Password != "password" {
		t.Errorf("expected persistence password to be 'password', got '%s'", *cfg.PersistenceConfig.Password)
	}

	if *cfg.PersistenceConfig.SSLMode != "require" {
		t.Errorf("expected persistence SSL mode to be 'require', got '%s'", *cfg.PersistenceConfig.SSLMode)
	}
}

func TestConfigPersistenceOmitDefault(t *testing.T) {
	data, err := os.ReadFile("./testconfig/testconfig_persistence_omit.json")
	if err != nil {
		t.Fatalf("failed to read test config: %v", err)
	}

	cfg := New()
	cfg.Load(data)
	if err := cfg.Parse(); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.PersistenceConfig.Driver != "sqlite" {
		t.Errorf("expected persistence driver to be 'sqlite', got '%s'", cfg.PersistenceConfig.Driver)
	}

	if cfg.PersistenceConfig.Address != "krb.db" {
		t.Errorf("expected persistence address to be 'krb.db', got '%s'", cfg.PersistenceConfig.Address)
	}
}
