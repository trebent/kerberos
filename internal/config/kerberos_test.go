package config

import (
	"os"
	"testing"

	"github.com/trebent/schemer"
)

func getSchemer(t *testing.T) *schemer.Schemer {
	t.Helper()
	s := schemer.New(getKerberosSchema(), getKerberosSupportingSchemas()...)
	return s
}

func checkErr(err error, t *testing.T) {
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestConfigBad(t *testing.T) {
	data, err := os.ReadFile("./testconfig/unknown_field.json")
	if err != nil {
		t.Fatalf("failed to read test config: %v", err)
	}

	sch := getSchemer(t)
	sch.Load(data)

	target := NewKerberos()
	if err := sch.Parse(target); err == nil {
		t.Fatalf("expected error when loading bad config, got nil")
	}
}

func TestConfigReferences(t *testing.T) {
	data, err := os.ReadFile("./testconfig/testconfig.json")
	if err != nil {
		t.Fatalf("failed to read test config: %v", err)
	}

	sch := getSchemer(t)
	sch.Load(data)

	target := NewKerberos()
	if err := sch.Parse(target); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	target.PostProcess()
}

func TestConfigAuth(t *testing.T) {
	t.Run("Basic auth with no API config", func(t *testing.T) {
		data, err := os.ReadFile("./testconfig/testconfig_auth_basic_no_api.json")
		if err != nil {
			t.Fatalf("failed to read test config: %v", err)
		}

		sch := getSchemer(t)
		sch.Load(data)

		target := NewKerberos()
		if err := sch.Parse(target); err != nil {
			t.Fatalf("failed to load config: %v", err)
		}
		target.PostProcess()

		if target.AuthCfg.Methods.Basic == nil {
			t.Fatalf("expected basic auth method to be non-nil, got nil")
		}

		if target.AuthCfg.Methods.Basic.API == nil {
			t.Fatalf("expected basic auth API config to be non-nil, got nil")
		}

		if target.AuthCfg.Methods.Basic.API.Cookies == nil {
			t.Fatalf("expected basic auth API cookies config to be non-nil, got nil")
		}

		if target.AuthCfg.Methods.Basic.API.Cookies.SameSite != "Strict" {
			t.Errorf("expected basic auth API cookies SameSite to be 'Strict', got '%s'", target.AuthCfg.Methods.Basic.API.Cookies.SameSite)
		}

		if target.AuthCfg.Methods.Basic.API.Origins == nil {
			t.Fatalf("expected basic auth API origins config to be non-nil, got nil")
		}
	})
}

func TestConfigAdmin(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		data, err := os.ReadFile("./testconfig/testconfig.json")
		if err != nil {
			t.Fatalf("failed to read test config: %v", err)
		}

		sch := getSchemer(t)
		sch.Load(data)

		target := NewKerberos()
		if err := sch.Parse(target); err != nil {
			t.Fatalf("failed to load config: %v", err)
		}
		target.PostProcess()

		if target.AdminCfg.SuperUser.ClientID != "admin" {
			t.Errorf("expected superuser client ID to be 'admin', got '%s'", target.AdminCfg.SuperUser.ClientID)
		}

		if target.AdminCfg.SuperUser.ClientSecret != "secret" {
			t.Errorf("expected superuser client secret to be 'secret', got '%s'", target.AdminCfg.SuperUser.ClientSecret)
		}
	})

	t.Run("Origins", func(t *testing.T) {
		data, err := os.ReadFile("./testconfig/testconfig_admin_origins.json")
		if err != nil {
			t.Fatalf("failed to read test config: %v", err)
		}

		sch := getSchemer(t)
		sch.Load(data)

		target := NewKerberos()
		if err := sch.Parse(target); err != nil {
			t.Fatalf("failed to load config: %v", err)
		}
		target.PostProcess()

		if target.AdminCfg.API.Origins == nil {
			t.Fatalf("expected admin origins config to be non-nil, got nil")
		}

		if len(target.AdminCfg.API.Origins.AllowedOrigins) != 2 {
			t.Errorf("expected admin origins allowed origins length to be 2, got %d", len(target.AdminCfg.API.Origins.AllowedOrigins))
		}

		if target.AdminCfg.API.Origins.AllowAll {
			t.Errorf("expected admin origins allow all to be false, got true")
		}
	})

	t.Run("Bad origins", func(t *testing.T) {
		data, err := os.ReadFile("./testconfig/testconfig_admin_bad_origins.json")
		if err != nil {
			t.Fatalf("failed to read test config: %v", err)
		}

		sch := getSchemer(t)
		sch.Load(data)

		target := NewKerberos()
		if err := sch.Parse(target); err == nil {
			t.Fatalf("expected error when loading config with bad origins, got nil")
		}
	})

	t.Run("Cookies", func(t *testing.T) {
		data, err := os.ReadFile("./testconfig/testconfig_admin_cookies.json")
		if err != nil {
			t.Fatalf("failed to read test config: %v", err)
		}

		sch := getSchemer(t)
		sch.Load(data)

		target := NewKerberos()
		if err := sch.Parse(target); err != nil {
			t.Fatalf("failed to load config: %v", err)
		}
		target.PostProcess()

		if target.AdminCfg.API.Cookies == nil {
			t.Fatalf("expected admin cookies config to be non-nil, got nil")
		}

		if target.AdminCfg.API.Cookies.Domain != "example.com" {
			t.Errorf("expected admin cookies domain to be example.com, got %s", target.AdminCfg.API.Cookies.Domain)
		}

		if target.AdminCfg.API.Cookies.SameSite != "Lax" {
			t.Errorf("expected admin cookies same site to be None, got %s", target.AdminCfg.API.Cookies.SameSite)
		}
	})
}

func TestConfigNoRouter(t *testing.T) {
	data, err := os.ReadFile("./testconfig/testconfig_nogw.json")
	if err != nil {
		t.Fatalf("failed to read test config: %v", err)
	}

	sch := getSchemer(t)
	sch.Load(data)

	target := NewKerberos()
	if err := sch.Parse(target); err == nil {
		t.Fatalf("expected error when loading config without router, got nil")
	}
}

func TestConfigEmpty(t *testing.T) {
	data, err := os.ReadFile("./testconfig/empty.json")
	if err != nil {
		t.Fatalf("failed to read test config: %v", err)
	}

	sch := getSchemer(t)
	sch.Load(data)

	target := NewKerberos()
	if err := sch.Parse(target); err == nil {
		t.Fatalf("expected error when loading empty config, got nil")
	}
}

func TestConfigOAS(t *testing.T) {
	data, err := os.ReadFile("./testconfig/testconfig_oas.json")
	if err != nil {
		t.Fatalf("failed to read test config: %v", err)
	}

	sch := getSchemer(t)
	sch.Load(data)

	target := NewKerberos()
	if err := sch.Parse(target); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	target.PostProcess()

	if len(target.OASCfg.Mappings) != 1 {
		t.Fatalf("expected 1 OAS mapping, got %d", len(target.OASCfg.Mappings))
	}

	mapping := target.OASCfg.Mappings[0]
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
	t.Run("Postgres happy", func(t *testing.T) {
		data, err := os.ReadFile("./testconfig/testconfig_persistence.json")
		if err != nil {
			t.Fatalf("failed to read test config: %v", err)
		}

		sch := getSchemer(t)
		sch.Load(data)

		target := NewKerberos()
		if err := sch.Parse(target); err != nil {
			t.Fatalf("failed to load config: %v", err)
		}
		target.PostProcess()

		if target.PersistenceCfg.Driver != "postgres" {
			t.Errorf("expected persistence driver to be 'postgres', got '%s'", target.PersistenceCfg.Driver)
		}

		if target.PersistenceCfg.Address != "localhost:5432" {
			t.Errorf("expected persistence address to be 'localhost', got '%s'", target.PersistenceCfg.Address)
		}

		if target.PersistenceCfg.Database != "kerberos" {
			t.Errorf("expected persistence database to be 'kerberos', got '%s'", target.PersistenceCfg.Database)
		}

		if *target.PersistenceCfg.Username != "user" {
			t.Errorf("expected persistence user to be 'user', got '%s'", *target.PersistenceCfg.Postgres.Username)
		}

		if *target.PersistenceCfg.Password != "password" {
			t.Errorf("expected persistence password to be 'password', got '%s'", *target.PersistenceCfg.Password)
		}

		if *target.PersistenceCfg.SSLMode != "require" {
			t.Errorf("expected persistence SSL mode to be 'require', got '%s'", *target.PersistenceCfg.SSLMode)
		}
	})

	t.Run("Default", func(t *testing.T) {
		data, err := os.ReadFile("./testconfig/testconfig_persistence_omit.json")
		if err != nil {
			t.Fatalf("failed to read test config: %v", err)
		}

		sch := getSchemer(t)
		sch.Load(data)

		target := NewKerberos()
		if err := sch.Parse(target); err != nil {
			t.Fatalf("failed to load config: %v", err)
		}
		target.PostProcess()

		if target.PersistenceCfg.Driver != "sqlite" {
			t.Errorf("expected persistence driver to be 'sqlite', got '%s'", target.PersistenceCfg.Driver)
		}

		if target.PersistenceCfg.Address != "krb.db" {
			t.Errorf("expected persistence address to be 'krb.db', got '%s'", target.PersistenceCfg.Address)
		}
	})

	t.Run("Postgres missing", func(t *testing.T) {
		data, err := os.ReadFile("./testconfig/testconfig_persistence_postgres_missing.json")
		if err != nil {
			t.Fatalf("failed to read test config: %v", err)
		}

		sch := getSchemer(t)
		sch.Load(data)

		target := NewKerberos()
		if err := sch.Parse(target); err == nil {
			t.Fatalf("expected error when loading config with missing postgres fields, got nil")
		}
	})
}

func TestConfigGateway(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		data, err := os.ReadFile("./testconfig/testconfig_gw.json")
		if err != nil {
			t.Fatalf("failed to read test config: %v", err)
		}

		sch := getSchemer(t)
		sch.Load(data)

		target := NewKerberos()
		if err := sch.Parse(target); err != nil {
			t.Fatalf("failed to load config: %v", err)
		}
		target.PostProcess()

		if target.GatewayCfg.Router == nil {
			t.Fatalf("expected router config to be non-nil, got nil")
		}

		if len(target.GatewayCfg.Router.Backends) != 1 {
			t.Fatalf("expected 1 backend, got %d", len(target.GatewayCfg.Router.Backends))
		}

		if target.GatewayCfg.TLS != nil {
			t.Errorf("expected TLS config to be nil, got non-nil")
		}

		if target.GatewayCfg.Router.Backends[0].Origins != nil {
			t.Fatalf("expected router backend's Origins config to be nil, got non-nil")
		}

		if target.GatewayCfg.Router.Backends[0].TimeoutMs != defaultCalloutTimeoutMs {
			t.Errorf("expected router backend TimeoutMs to be %d, got %d", defaultCalloutTimeoutMs, target.GatewayCfg.Router.Backends[0].TimeoutMs)
		}
	})

	t.Run("With TLS", func(t *testing.T) {
		data, err := os.ReadFile("./testconfig/testconfig_gw_tls.json")
		if err != nil {
			t.Fatalf("failed to read test config: %v", err)
		}

		sch := getSchemer(t)
		sch.Load(data)

		target := NewKerberos()
		if err := sch.Parse(target); err != nil {
			t.Fatalf("failed to load config: %v", err)
		}
		target.PostProcess()

		if target.GatewayCfg.TLS == nil {
			t.Fatalf("expected TLS config to be non-nil, got nil")
		}

		if target.GatewayCfg.TLS.CertFile != "/certs/server.crt" {
			t.Errorf("expected TLS ServerCertFile to be './certs/server.crt', got '%s'", target.GatewayCfg.TLS.CertFile)
		}

		if target.GatewayCfg.TLS.KeyFile != "/certs/server.key" {
			t.Errorf("expected TLS ServerKeyFile to be './certs/server.key', got '%s'", target.GatewayCfg.TLS.KeyFile)
		}
	})

	t.Run("Router valid backend TLS", func(t *testing.T) {
		data, err := os.ReadFile("./testconfig/testconfig_gw_router_tls.json")
		if err != nil {
			t.Fatalf("failed to read test config: %v", err)
		}

		sch := getSchemer(t)
		sch.Load(data)

		target := NewKerberos()
		if err := sch.Parse(target); err != nil {
			t.Fatalf("failed to load config: %v", err)
		}
		target.PostProcess()

		if target.GatewayCfg.Router == nil {
			t.Fatalf("expected router config to be non-nil, got nil")
		}

		if target.GatewayCfg.Router.Backends[0].TLS == nil {
			t.Fatalf("expected router backend's TLS config to be non-nil, got nil")
		}

		if target.GatewayCfg.Router.Backends[0].TLS.RootCAFile != "/certs/ca.crt" {
			t.Errorf("expected router backend TLS RootCAFile to be './certs/ca.crt', got '%s'", target.GatewayCfg.Router.Backends[0].TLS.RootCAFile)
		}

		if target.GatewayCfg.Router.Backends[0].TLS.ClientCertFile != "/certs/client.crt" {
			t.Errorf("expected router backend TLS ClientCertFile to be './certs/client.crt', got '%s'", target.GatewayCfg.Router.Backends[0].TLS.ClientCertFile)
		}

		if target.GatewayCfg.Router.Backends[0].TLS.ClientKeyFile != "/certs/client.key" {
			t.Errorf("expected router backend TLS ClientKeyFile to be './certs/client.key', got '%s'", target.GatewayCfg.Router.Backends[0].TLS.ClientKeyFile)
		}

		if target.GatewayCfg.Router.Backends[0].TLS.InsecureSkipVerify {
			t.Errorf("expected router backend TLS InsecureSkipVerify to be false, got true")
		}
	})

	t.Run("Router invalid backend TLS", func(t *testing.T) {
		data, err := os.ReadFile("./testconfig/testconfig_gw_router_tls_invalid.json")
		if err != nil {
			t.Fatalf("failed to read test config: %v", err)
		}

		sch := getSchemer(t)
		sch.Load(data)

		target := NewKerberos()
		if err := sch.Parse(target); err == nil {
			t.Fatalf("expected error when loading config with invalid router backend TLS, got nil")
		}
	})

	t.Run("Invalid backend name", func(t *testing.T) {
		data, err := os.ReadFile("./testconfig/testconfig_gw_router_invalid_backend_name.json")
		if err != nil {
			t.Fatalf("failed to read test config: %v", err)
		}

		sch := getSchemer(t)
		sch.Load(data)

		target := NewKerberos()
		if err := sch.Parse(target); err == nil {
			t.Fatalf("expected error when loading config with invalid router backend name, got nil")
		}
	})

	t.Run("Invalid backend port", func(t *testing.T) {
		data, err := os.ReadFile("./testconfig/testconfig_gw_router_invalid_backend_port.json")
		if err != nil {
			t.Fatalf("failed to read test config: %v", err)
		}

		sch := getSchemer(t)
		sch.Load(data)

		target := NewKerberos()
		if err := sch.Parse(target); err == nil {
			t.Fatalf("expected error when loading config with invalid router backend port, got nil")
		}
	})

	t.Run("Origins set", func(t *testing.T) {
		data, err := os.ReadFile("./testconfig/testconfig_gw_router_origins.json")
		if err != nil {
			t.Fatalf("failed to read test config: %v", err)
		}

		sch := getSchemer(t)
		sch.Load(data)

		target := NewKerberos()
		if err := sch.Parse(target); err != nil {
			t.Fatalf("failed to load config: %v", err)
		}
		target.PostProcess()

		if target.GatewayCfg.Router == nil {
			t.Fatalf("expected router config to be non-nil, got nil")
		}

		if target.GatewayCfg.Router.Backends[0].Origins == nil {
			t.Fatalf("expected router backend's Origins config to be non-nil, got nil")
		}

		if len(target.GatewayCfg.Router.Backends[0].Origins.AllowedOrigins) != 2 {
			t.Errorf("expected router backend Origins AllowedOrigins to have length 2, got %d", len(target.GatewayCfg.Router.Backends[0].Origins.AllowedOrigins))
		}

		if target.GatewayCfg.Router.Backends[0].Origins.AllowAll {
			t.Errorf("expected router backend Origins AllowAll to be false, got true")
		}
	})

	t.Run("Origins misconfigured", func(t *testing.T) {
		data, err := os.ReadFile("./testconfig/testconfig_gw_router_origins_invalid.json")
		if err != nil {
			t.Fatalf("failed to read test config: %v", err)
		}

		sch := getSchemer(t)
		sch.Load(data)

		target := NewKerberos()
		if err := sch.Parse(target); err == nil {
			t.Fatalf("expected error when loading config with misconfigured router backend Origins, got nil")
		}
	})
}
