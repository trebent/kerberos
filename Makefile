GRAFANA_PORT ?= 3000
PROM_PORT ?= 9090

build:
	@echo "\033[0;32mBuilding Kerberos...\033[0m"
	go build -o kerberos ./cmd/main.go
	@echo "\033[0;32mBuild complete.\033[0m"

validate:
	@echo "\033[0;32mValidating Kerberos...\033[0m"
	@go tool govulncheck ./...
	@go tool golangci-lint run
	@go test -v ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "\033[0;32mValidation complete.\033[0m"

setup-local-obs:
	@echo "\033[0;32mComposing local observability services...\033[0m"
	@GRAFANA_PORT=$(GRAFANA_PORT) PROM_PORT=$(PROM_PORT) docker compose -f observability/compose.yaml up -d --force-recreate
	@echo "\033[0;32mComposed local observability deployment.\033[0m"
	@echo " - Grafana on port:    $(GRAFANA_PORT)"
	@echo " - Prometheus on port: $(PROM_PORT)"

run:
	@echo "\033[0;32mRunning Kerberos...\033[0m"
	@LOG_TO_CONSOLE=1 LOG_VERBOSITY=100 OTEL_METRICS_EXPORTER=prometheus OTEL_TRACES_EXPORTER=otlp go run ./cmd/main.go
