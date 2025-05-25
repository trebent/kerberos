GRAFANA_PORT ?= 3000
PROM_PORT ?= 9090
KERBEROS_PORT ?= 30000

default: validate build

build:
	@echo "\033[0;32mBuilding Kerberos...\033[0m"
	go build -o kerberos ./cmd/kerberos/main.go
	@echo "\033[0;32mBuild complete.\033[0m"

validate:
	@echo "\033[0;32mValidating Kerberos...\033[0m"
	@go tool govulncheck ./...
	@golangci-lint run --fix
	@go test -v ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "\033[0;32mValidation complete.\033[0m"

setup-local-obs:
	@echo "\033[0;32mComposing local observability services...\033[0m"
	@GRAFANA_PORT=$(GRAFANA_PORT) PROM_PORT=$(PROM_PORT) docker compose -f observability/compose.yaml up -d --force-recreate
	@echo "\033[0;32mComposed local observability deployment.\033[0m"
	@echo " - Grafana on port:    $(GRAFANA_PORT)"
	@echo " - Prometheus on port: $(PROM_PORT)"

teardown-local-obs:
	@echo "\033[0;32mStopping local observability services...\033[0m"
	@docker compose -f observability/compose.yaml down
	@echo "\033[0;32mStopped local observability deployment.\033[0m"

run-krb-test:
	@echo "\033[0;32mRunning Kerberos for local tests...\033[0m"
	@LOG_TO_CONSOLE=1 LOG_VERBOSITY=100 OTEL_METRICS_EXPORTER=prometheus OTEL_TRACES_EXPORTER=otlp PORT=$(KERBEROS_PORT) TEST_ENDPOINT=1 go run ./cmd/kerberos/main.go

run-echo:
	@echo "\033[0;32mRunning echo server...\033[0m"
	@go run ./cmd/echo/main.go

generate-test-requests:
	@echo "\033[0;32mRunning some sample HTTP requests...\033[0m"
	curl -X GET -i localhost:$(KERBEROS_PORT)/test
	curl -X GET -i localhost:$(KERBEROS_PORT)/test?status_code=400
	curl -X PUT -i localhost:$(KERBEROS_PORT)/test
	curl -X PUT -i localhost:$(KERBEROS_PORT)/test?status_code=500
	curl -X POST -i localhost:$(KERBEROS_PORT)/test
	curl -X POST -i localhost:$(KERBEROS_PORT)/test?status_code=204
	curl -X PATCH -i localhost:$(KERBEROS_PORT)/test
	curl -X PATCH -i localhost:$(KERBEROS_PORT)/test?status_code=404
	curl -X DELETE -i localhost:$(KERBEROS_PORT)/test
	curl -X OPTIONS -i localhost:$(KERBEROS_PORT)/test
