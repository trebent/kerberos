package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

// ---- buildConfig ----

func TestBuildConfig_BasicBackend(t *testing.T) {
	t.Parallel()

	opts := &configOptions{
		backends:        []backendEntry{{name: "api", host: "localhost", port: 8080}},
		persistenceMode: driverSQLite,
	}

	data, err := buildConfig(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	gw, ok := result["gateway"].(map[string]any)
	if !ok {
		t.Fatal("missing gateway section")
	}

	router, ok := gw["router"].(map[string]any)
	if !ok {
		t.Fatal("missing router section")
	}

	backends, ok := router["backends"].([]any)
	if !ok || len(backends) != 1 {
		t.Fatalf("expected 1 backend, got %v", router["backends"])
	}
}

func TestBuildConfig_IncludesObs(t *testing.T) {
	t.Parallel()

	opts := &configOptions{
		backends:        []backendEntry{{name: "api", host: "api", port: 9000}},
		includeObs:      true,
		persistenceMode: driverSQLite,
	}

	data, _ := buildConfig(opts)

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if _, ok := result["observability"]; !ok {
		t.Error("expected observability section")
	}
}

func TestBuildConfig_PostgresPersistence(t *testing.T) {
	t.Parallel()

	opts := &configOptions{
		backends:        []backendEntry{{name: "svc", host: "svc", port: 80}},
		persistenceMode: driverPostgres,
	}

	data, _ := buildConfig(opts)

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	persistence, ok := result["persistence"].(map[string]any)
	if !ok {
		t.Fatal("missing persistence section")
	}

	if persistence["driver"] != driverPostgres {
		t.Errorf("expected driver=postgres, got %v", persistence["driver"])
	}
}

// ---- buildPrometheusYML ----

func TestBuildPrometheusYML_AllTargets(t *testing.T) {
	t.Parallel()

	opts := &obsConfigOptions{
		scrapeTargets: []string{"kerberos", "echo", "connector", "jaeger"},
	}

	yml := buildPrometheusYML(opts)

	for _, target := range opts.scrapeTargets {
		if !strings.Contains(yml, target) {
			t.Errorf("expected target %q in prometheus.yml", target)
		}
	}

	if !strings.Contains(yml, "kerberos:9464") {
		t.Error("expected kerberos:9464")
	}

	if !strings.Contains(yml, "echo:9463") {
		t.Error("expected echo:9463")
	}

	if !strings.Contains(yml, "connector:9462") {
		t.Error("expected connector:9462")
	}

	if !strings.Contains(yml, "jaeger:8888") {
		t.Error("expected jaeger:8888")
	}
}

func TestBuildPrometheusYML_SelectedTargets(t *testing.T) {
	t.Parallel()

	opts := &obsConfigOptions{
		scrapeTargets: []string{"kerberos"},
	}

	yml := buildPrometheusYML(opts)

	if !strings.Contains(yml, "kerberos:9464") {
		t.Error("expected kerberos:9464")
	}

	if strings.Contains(yml, "echo") {
		t.Error("echo should not be in output")
	}
}

// ---- buildGrafanaINI ----

func TestBuildGrafanaINI_Postgres(t *testing.T) {
	t.Parallel()

	opts := &obsConfigOptions{
		grafanaDB:        driverPostgres,
		grafanaAnonymous: true,
	}

	ini := buildGrafanaINI(opts)

	if !strings.Contains(ini, "type = postgres") {
		t.Error("expected type = postgres")
	}

	if !strings.Contains(ini, "host = postgres:5432") {
		t.Error("expected host = postgres:5432")
	}

	if !strings.Contains(ini, "enabled = true") {
		t.Error("expected enabled = true in auth.anonymous")
	}
}

func TestBuildGrafanaINI_SQLite_NoAnon(t *testing.T) {
	t.Parallel()

	opts := &obsConfigOptions{
		grafanaDB:        "sqlite3",
		grafanaAnonymous: false,
	}

	ini := buildGrafanaINI(opts)

	if !strings.Contains(ini, "type = sqlite3") {
		t.Error("expected type = sqlite3")
	}

	if strings.Contains(ini, "host =") {
		t.Error("sqlite config should not contain host")
	}

	if !strings.Contains(ini, "enabled = false") {
		t.Error("expected enabled = false in auth.anonymous")
	}
}

// ---- buildGrafanaDatasourcesYML ----

func TestBuildGrafanaDatasourcesYML(t *testing.T) {
	t.Parallel()

	yml := buildGrafanaDatasourcesYML()

	if !strings.Contains(yml, "prometheus") {
		t.Error("expected prometheus datasource")
	}

	if !strings.Contains(yml, "http://prometheus:9090") {
		t.Error("expected prometheus URL")
	}
}

// ---- buildGrafanaDashboardsYML ----

func TestBuildGrafanaDashboardsYML(t *testing.T) {
	t.Parallel()

	yml := buildGrafanaDashboardsYML()

	for _, dashboard := range []string{"prometheus.json", "kerberos_runtime.json", "kerberos_http.json"} {
		if !strings.Contains(yml, dashboard) {
			t.Errorf("expected dashboard %q in grafana-dashboards.yml", dashboard)
		}
	}
}

// ---- buildJaegerYML ----

func TestBuildJaegerYML(t *testing.T) {
	t.Parallel()

	yml := buildJaegerYML()

	if !strings.Contains(yml, "otlp") {
		t.Error("expected otlp receiver in jaeger.yml")
	}

	if !strings.Contains(yml, "badger_store") {
		t.Error("expected badger_store backend")
	}

	if !strings.Contains(yml, "16686") {
		t.Error("expected jaeger query port 16686")
	}
}

// ---- buildConnectorJSON ----

func TestBuildConnectorJSON_SQLite(t *testing.T) {
	t.Parallel()

	opts := &connectorOptions{
		corsOrigin:      "http://localhost:3000",
		persistenceMode: driverSQLite,
	}

	data, err := buildConnectorJSON(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	origins, ok := result["origins"].(map[string]any)
	if !ok {
		t.Fatal("missing origins section")
	}

	allowed, ok := origins["allowedOrigins"].([]any)
	if !ok || len(allowed) == 0 {
		t.Fatal("expected at least one allowed origin")
	}

	if allowed[0] != "http://localhost:3000" {
		t.Errorf("expected origin http://localhost:3000, got %v", allowed[0])
	}

	persistence, ok := result["persistence"].(map[string]any)
	if !ok {
		t.Fatal("missing persistence section")
	}

	if persistence["driver"] != driverSQLite {
		t.Errorf("expected driver=sqlite, got %v", persistence["driver"])
	}
}

func TestBuildConnectorJSON_DefaultOrigin(t *testing.T) {
	t.Parallel()

	opts := &connectorOptions{
		corsOrigin:      "",
		persistenceMode: driverSQLite,
	}

	data, err := buildConnectorJSON(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(string(data), "http://kerberos:30001") {
		t.Error("expected default origin http://kerberos:30001")
	}
}
