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

func TestBuildConfig_AlwaysIncludesObs(t *testing.T) {
	t.Parallel()

	opts := &configOptions{
		backends:        []backendEntry{{name: "api", host: "api", port: 9000}},
		persistenceMode: driverSQLite,
	}

	data, _ := buildConfig(opts)

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if _, ok := result["observability"]; !ok {
		t.Error("expected observability section to always be present")
	}
}

func TestBuildConfig_PerBackendAuth(t *testing.T) {
	t.Parallel()

	opts := &configOptions{
		backends: []backendEntry{
			{name: "secured", host: "secured", port: 8080, auth: true},
			{name: "open", host: "open", port: 8081, auth: false},
		},
		persistenceMode: driverSQLite,
	}

	data, _ := buildConfig(opts)

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	auth, ok := result["auth"].(map[string]any)
	if !ok {
		t.Fatal("expected auth section")
	}

	scheme, _ := auth["scheme"].(map[string]any)
	mappings, ok := scheme["mappings"].([]any)
	if !ok || len(mappings) != 1 {
		t.Fatalf("expected exactly 1 auth mapping, got %v", scheme["mappings"])
	}

	mapping, _ := mappings[0].(map[string]any)
	if mapping["backend"] != "secured" {
		t.Errorf("expected auth mapping for 'secured', got %v", mapping["backend"])
	}
}

func TestBuildConfig_NoAuthWhenNoneEnabled(t *testing.T) {
	t.Parallel()

	opts := &configOptions{
		backends:        []backendEntry{{name: "open", host: "open", port: 8080, auth: false}},
		persistenceMode: driverSQLite,
	}

	data, _ := buildConfig(opts)

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if _, ok := result["auth"]; ok {
		t.Error("expected no auth section when no backend has auth enabled")
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

// ---- resolveScrapeJobs / buildPrometheusYML ----

func TestResolveScrapeJobs_BackendsAndConnector(t *testing.T) {
	t.Parallel()

	opts := &configOptions{
		backends: []backendEntry{
			{name: "echo", host: "echo", port: 15000},
			{name: "api", host: "api-host", port: 8080},
		},
		includeConnector: true,
		obsOpts: obsConfigOptions{
			scrapeTargets: []string{"kerberos", "jaeger", "echo", "api"},
		},
	}

	yml := buildPrometheusYML(resolveScrapeJobs(opts))

	for _, want := range []string{
		"kerberos:9464",
		"jaeger:8888",
		"echo:9464",
		"api-host:9464",
		"connector:9464",
	} {
		if !strings.Contains(yml, want) {
			t.Errorf("expected %q in prometheus.yml, got:\n%s", want, yml)
		}
	}
}

func TestResolveScrapeJobs_NoEchoWhenNotRegistered(t *testing.T) {
	t.Parallel()

	opts := &configOptions{
		backends: []backendEntry{{name: "api", host: "api", port: 8080}},
		obsOpts: obsConfigOptions{
			scrapeTargets: []string{"kerberos", "echo", "api"},
		},
	}

	yml := buildPrometheusYML(resolveScrapeJobs(opts))

	if strings.Contains(yml, "echo") {
		t.Errorf("echo should not appear when it was not registered as a backend:\n%s", yml)
	}

	if !strings.Contains(yml, "api:9464") {
		t.Error("expected api:9464")
	}
}

func TestResolveScrapeJobs_ConnectorOnlyWhenIncluded(t *testing.T) {
	t.Parallel()

	opts := &configOptions{
		includeConnector: false,
		obsOpts:          obsConfigOptions{scrapeTargets: []string{"kerberos"}},
	}

	yml := buildPrometheusYML(resolveScrapeJobs(opts))

	if strings.Contains(yml, "connector") {
		t.Errorf("connector should not be scraped when not included:\n%s", yml)
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

// ---- buildCompose ----

func TestBuildCompose_NoEnvVarInjection(t *testing.T) {
	t.Parallel()

	opts := &composeOptions{
		includeEcho:      true,
		includeObsStack:  true,
		includePostgres:  false,
		includeConnector: true,
	}

	out := buildCompose(opts)

	if strings.Contains(out, "${") {
		t.Error("compose output should not contain any ${...} env var injection")
	}

	for _, want := range []string{
		"ghcr.io/trebent/kerberos:latest",
		"- 30000:30000",
		"- 30001:30001",
		"- 3000:3000",
		"- 16686:16686",
		"LOG_VERBOSITY=0",
		"krbdata:/data",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected compose output to contain %q", want)
		}
	}

	// Only kerberos gw/admin, grafana and jaeger publish host ports.
	for _, notWant := range []string{
		"- 9464:9464",
		"- 15000:15000",
		"- 30100:30100",
		"- 9090:9090",
		"LOG_VERBOSITY=20",
	} {
		if strings.Contains(out, notWant) {
			t.Errorf("compose output should not contain %q", notWant)
		}
	}
}

func TestBuildCompose_SharedSqliteVolumeOmittedWithPostgres(t *testing.T) {
	t.Parallel()

	out := buildCompose(&composeOptions{includeConnector: true, includePostgres: true})

	if strings.Contains(out, "krbdata") {
		t.Errorf("krbdata volume should not be present when Postgres is enabled:\n%s", out)
	}
}

// ---- buildConnectorJSON ----

func TestBuildConnectorJSON_SQLite(t *testing.T) {
	t.Parallel()

	opts := &connectorOptions{
		allowAllOrigins: true,
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

	if origins["allowAll"] != true {
		t.Errorf("expected allowAll=true, got %v", origins["allowAll"])
	}

	if _, ok := origins["denyAll"]; ok {
		t.Error("did not expect denyAll when allowAll is set")
	}

	persistence, ok := result["persistence"].(map[string]any)
	if !ok {
		t.Fatal("missing persistence section")
	}

	if persistence["driver"] != driverSQLite {
		t.Errorf("expected driver=sqlite, got %v", persistence["driver"])
	}

	if persistence["address"] != sqliteSharedPath {
		t.Errorf("expected shared sqlite address %q, got %v", sqliteSharedPath, persistence["address"])
	}
}

func TestBuildConnectorJSON_DenyAll(t *testing.T) {
	t.Parallel()

	opts := &connectorOptions{
		allowAllOrigins: false,
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

	origins, _ := result["origins"].(map[string]any)
	if origins["denyAll"] != true {
		t.Errorf("expected denyAll=true, got %v", origins["denyAll"])
	}

	if _, ok := origins["allowAll"]; ok {
		t.Error("did not expect allowAll when denyAll is set")
	}
}
