# User config
SUPERUSER_CLIENT_ID ?= admin
SUPERUSER_CLIENT_SECRET ?= secret
ADMIN_USER_ALWAYS ?= always
ADMIN_USER_ALWAYS_PASSWORD ?= password123
AUTH_BASIC_USER_ALWAYS ?= always
AUTH_BASIC_USER_ALWAYS_PASSWORD ?= password123

# Port config
GRAFANA_PORT ?= 3000
PROM_PORT ?= 9090
KERBEROS_PORT ?= 30000
KERBEROS_ADMIN_PORT ?= 30001
KERBEROS_METRICS_PORT ?= 9464
ECHO_PORT ?= 15000
ECHO_METRICS_PORT ?= 9463
CONNECTOR_PORT ?= 30100
CONNECTOR_METRICS_PORT ?= 9462

# Docker network config
NETWORK_NAME ?= kerberos

BOLD_RED=\033[1;31m
BOLD_GREEN=\033[1;32m
BOLD_YELLOW=\033[1;33m
BOLD_BLUE=\033[1;34m

RED=\033[31m
GREEN=\033[32m
YELLOW=\033[33m
BLUE=\033[34m
RESET=\033[0m

LOG_VERBOSITY ?= 20
VERSION ?= $(shell git describe --tags --always)
GOPATH ?= $(shell go env GOPATH)
GOBIN ?= $(GOPATH)/bin

define cecho
@printf "${2}${1}${RESET}\n"
endef

default: static-analysis/lint static-analysis/vulncheck build test/unit postgres/run test/unit/postgres postgres/stop

build:
	$(call cecho,Building Kerberos binary...,$(BOLD_YELLOW))
	@mkdir -p build
	CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o build/kerberos .

clean:
	@rm -rf build/

codegen: install/deps
	$(call cecho,Running codegen for Kerberos...,$(BOLD_YELLOW))
	@go generate ./...

	$(call cecho,Running codegen for integration tests...,$(BOLD_YELLOW))
	@cd test/suites/integration && go generate ./...

run:
	$(call cecho,Running Kerberos...,$(BOLD_YELLOW))
	mkdir -p build
	PORT=$(KERBEROS_PORT) \
	ADMIN_PORT=$(KERBEROS_ADMIN_PORT) \
	LOG_TO_CONSOLE=true \
	LOG_VERBOSITY=$(LOG_VERBOSITY) \
	OAS_DIRECTORY=$(PWD)/openapi \
	VERSION=$(VERSION) \
	go run . --config ./test/config/local.json

compose/clean:
	$(call cecho,Cleaning up Kerberos test environment...,$(BOLD_YELLOW))
	@docker compose -f test/compose/integration/compose.yaml down --volumes --remove-orphans
	@docker compose -f test/compose/security/compose.yaml down --volumes --remove-orphans

compose/up:
	$(call cecho,Composing Kerberos test environment...,$(BOLD_YELLOW))
	@VERSION=$(VERSION) \
	KERBEROS_PORT=$(KERBEROS_PORT) \
	KERBEROS_ADMIN_PORT=$(KERBEROS_ADMIN_PORT) \
	KERBEROS_METRICS_PORT=$(KERBEROS_METRICS_PORT) \
	LOG_VERBOSITY=$(LOG_VERBOSITY) \
	PROM_PORT=$(PROM_PORT) \
	GRAFANA_PORT=$(GRAFANA_PORT) \
	ECHO_PORT=$(ECHO_PORT) \
	ECHO_METRICS_PORT=$(ECHO_METRICS_PORT) \
	docker compose -f test/compose/integration/compose.yaml up -d --force-recreate
	$(call cecho,Waiting for Kerberos to be ready...,$(BOLD_YELLOW))
	@until [ "$$(curl -s -o /dev/null -w '%{http_code}' localhost:$(KERBEROS_ADMIN_PORT)/api/admin/flow)" = "401" ]; do \
	echo "Waiting for Kerberos admin API..."; \
	sleep 1; \
	done; \
	echo "Kerberos is ready!"

compose/down:
	$(call cecho,Tearing down Kerberos test environment...,$(BOLD_YELLOW))
	@docker compose -f test/compose/integration/compose.yaml down

compose/logs:
	@docker compose -f test/compose/integration/compose.yaml logs kerberos echo protected-echo

compose/logs/follow:
	@docker compose -f test/compose/integration/compose.yaml logs -f kerberos echo protected-echo

compose/ps:
	@docker compose -f test/compose/integration/compose.yaml ps

compose/security/logs:
	@docker compose -f test/compose/security/compose.yaml logs kerberos echo

compose/security/logs/follow:
	@docker compose -f test/compose/security/compose.yaml logs kerberos echo -f

compose/security/up:
	$(call cecho,Composing Kerberos security test environment...,$(BOLD_YELLOW))
	@VERSION=$(VERSION) \
	KERBEROS_PORT=$(KERBEROS_PORT) \
	KERBEROS_ADMIN_PORT=$(KERBEROS_ADMIN_PORT) \
	LOG_VERBOSITY=$(LOG_VERBOSITY) \
	ECHO_PORT=$(ECHO_PORT) \
	ECHO_METRICS_PORT=$(ECHO_METRICS_PORT) \
	docker compose -f test/compose/security/compose.yaml up -d --force-recreate
	@until [ "$$(curl -s -o /dev/null -w '%{http_code}' --cacert test/certs/ca.crt https://localhost:$(KERBEROS_ADMIN_PORT)/api/admin/flow)" = "401" ]; do \
	echo "Waiting for Kerberos admin API..."; \
	sleep 1; \
	done; \
	echo "Kerberos is ready!"

compose/security/down:
	$(call cecho,Tearing down Kerberos security test environment...,$(BOLD_YELLOW))
	@docker compose -f test/compose/security/compose.yaml down

compose/connector/up:
	$(call cecho,Composing Kerberos Admin Connector test environment...,$(BOLD_YELLOW))
	@VERSION=$(VERSION) \
	KERBEROS_PORT=$(KERBEROS_PORT) \
	KERBEROS_ADMIN_PORT=$(KERBEROS_ADMIN_PORT) \
	CONNECTOR_PORT=$(CONNECTOR_PORT) \
	ECHO_PORT=$(ECHO_PORT) \
	LOG_VERBOSITY=$(LOG_VERBOSITY) \
	docker compose -f test/compose/connector/compose.yaml up -d --force-recreate
	@until [ "$$(curl -s -o /dev/null -w '%{http_code}' localhost:$(CONNECTOR_PORT))" = "401" ]; do \
	echo "Waiting for Admin Connector API..."; \
	sleep 1; \
	done; \
	echo "Admin Connector is ready!"

compose/connector/down:
	$(call cecho,Tearing down Kerberos Admin Connector test environment...,$(BOLD_YELLOW))
	@docker compose -f test/compose/connector/compose.yaml down

compose/connector/logs:
	@docker compose -f test/compose/connector/compose.yaml logs kerberos echo connector

compose/connector/logs/follow:
	@docker compose -f test/compose/connector/compose.yaml logs -f kerberos echo connector

coverage:
	@go tool cover -html=build/coverage.out -o build/coverage.html
	@go tool cover -func=build/coverage.out | awk 'END {print $$3}'

coverage/report:
	$(call cecho,Generating coverage report for Kerberos...,$(BOLD_YELLOW))
	@go tool cover -html=build/coverage.out -o build/coverage.html
	@echo "### Code Coverage: $$(go tool cover -func=build/coverage.out | awk '/^total:/{print $$3}')"

docker/build:
	$(call cecho,Building Kerberos Docker image...,$(BOLD_YELLOW))
	docker build --build-arg VERSION=$(VERSION) -t ghcr.io/trebent/kerberos:$(VERSION) -f docker/krb.Dockerfile .

docker/logs:
	@docker logs kerberos -f

docker/rm:
	@docker rm kerberos || true

poc/build:
	$(call cecho,Building Kerberos Docker image for PoC...,$(BOLD_YELLOW))
	docker build --build-arg VERSION=$(VERSION) --target poc-runtime -t ghcr.io/trebent/kerberos:$(VERSION) -f docker/krb.Dockerfile .

poc/run:
	docker run -d --name kerberos ghcr.io/trebent/kerberos:$(VERSION) --config /poc.json

docker/run: docker/build docker/stop docker/rm docker/network/create
	$(call cecho,Running Kerberos Docker container...,$(BOLD_YELLOW))
	docker run -d \
	-p $(KERBEROS_PORT):$(KERBEROS_PORT) \
	-p $(KERBEROS_ADMIN_PORT):$(KERBEROS_ADMIN_PORT) \
	-p $(KERBEROS_METRICS_PORT):$(KERBEROS_METRICS_PORT) \
	-e PORT=$(KERBEROS_PORT) \
	-e ADMIN_PORT=$(KERBEROS_ADMIN_PORT) \
	-e LOG_TO_CONSOLE=true \
	-e LOG_VERBOSITY=$(LOG_VERBOSITY) \
	-e OTEL_EXPORTER_PROMETHEUS_HOST=0.0.0.0 \
	-e OTEL_METRICS_EXPORTER=prometheus \
	-e OTEL_EXPORTER_PROMETHEUS_PORT=$(KERBEROS_METRICS_PORT) \
	-v $(PWD)/test/config:/config:ro \
	-v $(PWD)/test/oas:/oas:ro \
	--network $(NETWORK_NAME) \
	--name kerberos \
	ghcr.io/trebent/kerberos:$(VERSION) \
	--config /config/docker.json

docker/stop:
	@docker stop kerberos || true

docker/network/create:
	@docker network inspect $(NETWORK_NAME) > /dev/null 2>&1 || docker network create $(NETWORK_NAME)

docker/network/rm:
	@docker network rm $(NETWORK_NAME) || true

echo/build:
	$(call cecho,Building Echo binary...,$(BOLD_YELLOW))
	@CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o build/echo ./cmd/echo

echo/docker/build:
	$(call cecho,Building Echo Docker image...,$(BOLD_YELLOW))
	@docker build --build-arg VERSION=$(VERSION) \
	-f docker/echo.Dockerfile \
	-t ghcr.io/trebent/kerberos/echo:$(VERSION) \
	.

echo/docker/logs:
	@docker logs echo

echo/docker/rm:
	@docker rm echo || true

echo/docker/run: echo/docker/build echo/docker/stop echo/docker/rm docker/network/create
	$(call cecho,Running Echo Docker container...,$(BOLD_YELLOW))
	@docker run -d \
	-p $(ECHO_PORT):$(ECHO_PORT) \
	-p $(ECHO_METRICS_PORT):$(ECHO_METRICS_PORT) \
	-e LOG_VERBOSITY=$(LOG_VERBOSITY) \
	-e OTEL_METRICS_EXPORTER=prometheus \
	-e OTEL_EXPORTER_PROMETHEUS_HOST=0.0.0.0 \
	-e OTEL_EXPORTER_PROMETHEUS_PORT=$(ECHO_METRICS_PORT) \
	-e OTEL_TRACES_EXPORTER=none \
	-e VERSION=$(VERSION) \
	--network $(NETWORK_NAME) \
	--name echo \
	ghcr.io/trebent/kerberos/echo:$(VERSION)

echo/docker/stop:
	@docker stop echo || true

echo/run:
	$(call cecho,Running echo...,$(BOLD_YELLOW))
	VERSION=$(VERSION) \
	go run ./cmd/echo

connector/build:
	$(call cecho,Building Admin Connector binary...,$(BOLD_YELLOW))
	@mkdir -p build
	@CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o build/connector ./cmd/admin-connector

connector/run:
	$(call cecho,Running Admin Connector...,$(BOLD_YELLOW))
	VERSION=$(VERSION) \
	TARGET=localhost:$(ECHO_PORT) \
	LOG_TO_CONSOLE=true \
	go run ./cmd/admin-connector --config test/config/connector/local.json

connector/docker/build:
	$(call cecho,Building Admin Connector Docker image...,$(BOLD_YELLOW))
	@docker build --build-arg VERSION=$(VERSION) \
	-f docker/connector.Dockerfile \
	-t ghcr.io/trebent/kerberos/admin-connector:$(VERSION) \
	.

connector/docker/run: connector/docker/build connector/docker/stop connector/docker/rm docker/network/create
	$(call cecho,Running Admin Connector Docker container...,$(BOLD_YELLOW))
	@docker run -d \
	-p $(CONNECTOR_PORT):$(CONNECTOR_PORT) \
	-p $(CONNECTOR_METRICS_PORT):$(CONNECTOR_METRICS_PORT) \
	-e VERSION=$(VERSION) \
	-e TARGET=echo:$(ECHO_PORT) \
	-e LOG_TO_CONSOLE=true \
	-e LOG_VERBOSITY=$(LOG_VERBOSITY) \
	-e OTEL_METRICS_EXPORTER=prometheus \
	-e OTEL_EXPORTER_PROMETHEUS_HOST=0.0.0.0 \
	-e OTEL_EXPORTER_PROMETHEUS_PORT=$(CONNECTOR_METRICS_PORT) \
	-e OTEL_TRACES_EXPORTER=none \
	-v $(PWD)/test/config/connector:/config:ro \
	--network $(NETWORK_NAME) \
	--name connector \
	ghcr.io/trebent/kerberos/admin-connector:$(VERSION) \
	--config /config/docker.json

connector/docker/stop:
	@docker stop connector || true

connector/docker/rm:
	@docker rm connector || true

install/deps:
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.6.0

install/lint:
	curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(GOBIN) v2.12.2

# This uses the integration test suite to provision Kerberos with test data created by the main entrypoint of the integration test suite.
krb/provision:
	$(call cecho,Provisioning Kerberos with test data...,$(BOLD_YELLOW))
	@cd test/suites/integration && go test -v ./... -count=1 -failfast -run NotExist

krb/superuser-login:
	$(call cecho,Logging in with basic auth to Kerberos...,$(BOLD_YELLOW))
	curl -s -o /dev/null -D - -X POST localhost:$(KERBEROS_ADMIN_PORT)/api/admin/superuser/login \
		-H "Content-Type: application/json" \
		-d '{"clientId":"$(SUPERUSER_CLIENT_ID)","clientSecret":"$(SUPERUSER_CLIENT_SECRET)"}'
	
krb/admin-login:
	$(call cecho,Logging in with basic auth to Kerberos...,$(BOLD_YELLOW))
	curl -s -o /dev/null -D - -X POST localhost:$(KERBEROS_ADMIN_PORT)/api/admin/login \
		-H "Content-Type: application/json" \
		-d '{"username":"$(ADMIN_USER_ALWAYS)","password":"$(ADMIN_USER_ALWAYS_PASSWORD)"}'

krb/basic-auth-login:
	$(call cecho,Logging in with basic auth to Kerberos...,$(BOLD_YELLOW))
	curl -s -o /dev/null -D - -X POST localhost:$(KERBEROS_ADMIN_PORT)/api/auth/basic/organisations/1/login \
		-H "Content-Type: application/json" \
		-d '{"username":"$(AUTH_BASIC_USER_ALWAYS)","password":"$(AUTH_BASIC_USER_ALWAYS_PASSWORD)"}'

postgres/run:
	$(call cecho,Running PostgreSQL for Kerberos...,$(BOLD_YELLOW))
	@docker run -d \
	--rm \
	-p 5432:5432 \
	-e POSTGRES_USER=kerberos \
	-e POSTGRES_PASSWORD=kerberos \
	-e POSTGRES_DB=kerberos \
	--name kerberos-postgres \
	postgres:18.4-alpine3.23
	$(call cecho,Waiting for PostgreSQL to be ready...,$(BOLD_YELLOW))
	@until docker exec kerberos-postgres pg_isready -U kerberos -d kerberos > /dev/null 2>&1; do \
	echo "Waiting for PostgreSQL..."; \
	sleep 1; \
	done; \
	echo "PostgreSQL is ready!"

postgres/stop:
	$(call cecho,Stopping PostgreSQL for Kerberos...,$(BOLD_YELLOW))
	@docker stop kerberos-postgres || true

static-analysis/lint:
	$(call cecho,Running linter for Kerberos...,$(BOLD_YELLOW))
	@golangci-lint run --fix

static-analysis/vulncheck:
	$(call cecho,Running vulnerability check for Kerberos...,$(BOLD_YELLOW))
	@go tool -modfile=./tools/go.mod govulncheck ./...

static-analysis/vulncheck/sarif:
	$(call cecho,Running vulnerability check for Kerberos...,$(BOLD_YELLOW))
	@mkdir -p build
	@go tool -modfile=./tools/go.mod govulncheck -format sarif ./... > build/govulncheck-report.sarif

test/echo:
	$(call cecho,Sending a test request to echo...,$(BOLD_YELLOW))
	curl -X GET -i localhost:$(KERBEROS_PORT)/gw/backend/echo/hi

test/echo-methods:
	$(call cecho,Generating test HTTP requests for the echo backend...,$(BOLD_YELLOW))
	curl -X GET -i localhost:$(KERBEROS_PORT)/gw/backend/echo/hi
	curl -X PUT -i localhost:$(KERBEROS_PORT)/gw/backend/echo/hi
	curl -X POST -i localhost:$(KERBEROS_PORT)/gw/backend/echo/hi
	curl -X PATCH -i localhost:$(KERBEROS_PORT)/gw/backend/echo/hi
	curl -X DELETE -i localhost:$(KERBEROS_PORT)/gw/backend/echo/hi
	curl -X OPTIONS -i localhost:$(KERBEROS_PORT)/gw/backend/echo/hi

test/protected-echo:
	$(call cecho,Sending a test request to protected-echo...,$(BOLD_YELLOW))
	curl -X GET -i localhost:$(KERBEROS_PORT)/gw/backend/protected-echo/hi

test/integration:
	$(call cecho,Running integration tests for Kerberos...,$(BOLD_YELLOW))
	@cd test/suites/integration && go test -v ./... -count=1 -failfast

test/integration/json:
	$(call cecho,Running integration tests for Kerberos...,$(BOLD_YELLOW))
	@mkdir -p build
	@cd test/suites/integration && go test -v -json ./... -count=1 -failfast > $(CURDIR)/build/integration-test-output.json

test/security:
	$(call cecho,Running security tests for Kerberos...,$(BOLD_YELLOW))
	@cd test/suites/security && go test -v ./... -count=1 -failfast

test/security/json:
	$(call cecho,Running security tests for Kerberos...,$(BOLD_YELLOW))
	@mkdir -p build
	@cd test/suites/security && go test -v -json ./... -count=1 -failfast > $(CURDIR)/build/security-test-output.json

test/connector:
	$(call cecho,Running Admin Connector tests for Kerberos...,$(BOLD_YELLOW))
	@cd test/suites/connector && go test -v ./... -count=1 -failfast

test/connector/json:
	$(call cecho,Running Admin Connector tests for Kerberos...,$(BOLD_YELLOW))
	@mkdir -p build
	@cd test/suites/connector && go test -v -json ./... -count=1 -failfast > $(CURDIR)/build/connector-test-output.json

test/unit:
	$(call cecho,Running unit tests for Kerberos...,$(BOLD_YELLOW))
	@mkdir -p build
	@go test -v ./... -timeout 20s -failfast -coverprofile=build/coverage.out -covermode=atomic

test/unit/json:
	$(call cecho,Running unit tests for Kerberos...,$(BOLD_YELLOW))
	@mkdir -p build
	@go test -v -json -coverprofile=build/coverage.out -covermode=atomic ./... -timeout 20s -failfast > build/unit-test-output.json

# admin tests run with -p 1 since there are two main appliers of the same schema.
test/unit/postgres:
	$(call cecho,Running unit tests (admin, basic auth) for Kerberos with PostgreSQL...,$(BOLD_YELLOW))
	cd internal/admin && go test -v -p 1 ./... -timeout 20s -failfast -tags=postgres_integration
	cd internal/auth/method/basic && go test -v ./... -timeout 20s -failfast -tags=postgres_integration
	cd internal/db/postgres && go test -v ./... -timeout 20s -failfast -tags=postgres_integration

validate: static-analysis/lint test/unit static-analysis/vulncheck
	$(call cecho,Static analysis complete.,$(BOLD_GREEN))

version:
	$(call cecho,Kerberos version: $(VERSION),$(BOLD_BLUE))
