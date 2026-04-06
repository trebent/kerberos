GRAFANA_PORT ?= 3000
PROM_PORT ?= 9090
KERBEROS_PORT ?= 30000
KERBEROS_ADMIN_PORT ?= 30001
SUPERUSER_CLIENT_ID ?= admin
SUPERUSER_CLIENT_SECRET ?= secret
KERBEROS_METRICS_PORT ?= 9464
ECHO_PORT ?= 15000
ECHO_METRICS_PORT ?= 9463

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

define cecho
	@echo -e "${2}${1}${RESET}"
endef

default: lint vulncheck go-build unittest

clean:
	@rm -rf build/

version:
	$(call cecho,Kerberos version: $(VERSION),$(BOLD_BLUE))

lint:
	$(call cecho,Running linter for Kerberos...,$(BOLD_YELLOW))
	@golangci-lint run --fix

codegen:
	$(call cecho,Running codegen for Kerberos...,$(BOLD_YELLOW))
	@go generate ./...
	
	$(call cecho,Running codegen for integration tests...,$(BOLD_YELLOW))
	@cd test/integration && go generate ./...

unittest:
	$(call cecho,Running unit tests for Kerberos...,$(BOLD_YELLOW))
	@mkdir -p build
	@go test -v ./... -timeout 20s -failfast -coverprofile=build/coverage.out -covermode=atomic

unittest-json:
	$(call cecho,Running unit tests for Kerberos...,$(BOLD_YELLOW))
	@mkdir -p build
	@go test -v -json -coverprofile=build/coverage.out -covermode=atomic ./... -timeout 20s -failfast > build/unit-test-output.json

coverage-report:
	$(call cecho,Generating coverage report for Kerberos...,$(BOLD_YELLOW))
	@go tool cover -html=build/coverage.out -o build/coverage.html
	@echo "### Code Coverage: $$(go tool cover -func=build/coverage.out | awk '/^total:/{print $$3}')"

coverage:
	@go tool cover -html=build/coverage.out -o build/coverage.html
	@go tool cover -func=build/coverage.out | awk 'END {print $$3}'

integrationtest:
	$(call cecho,Running integration tests for Kerberos...,$(BOLD_YELLOW))
	@cd test/integration && go test -v ./... -count=1 -failfast

integrationtest-json:
	$(call cecho,Running integration tests for Kerberos...,$(BOLD_YELLOW))
	@mkdir -p build
	@cd test/integration && go test -v -json ./... -count=1 -failfast > $(CURDIR)/build/integration-test-output.json

vulncheck:
	$(call cecho,Running vulnerability check for Kerberos...,$(BOLD_YELLOW))
	@go tool -modfile=./tools/go.mod govulncheck ./...

vulncheck-sarif:
	$(call cecho,Running vulnerability check for Kerberos...,$(BOLD_YELLOW))
	@mkdir -p build
	@go tool -modfile=./tools/go.mod govulncheck -format sarif ./... > build/govulncheck-report.sarif

staticcheck: lint unittest vulncheck
	$(call cecho,Static analysis complete.,$(BOLD_GREEN))

go-build:
	$(call cecho,Building Kerberos binary...,$(BOLD_YELLOW))
	@mkdir -p build
	CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o build/kerberos .

run:
	$(call cecho,Running Kerberos...,$(BOLD_YELLOW))
	mkdir -p build
	PORT=$(KERBEROS_PORT) \
		ADMIN_PORT=$(KERBEROS_ADMIN_PORT) \
		LOG_TO_CONSOLE=true \
		LOG_VERBOSITY=$(LOG_VERBOSITY) \
		DB_DIRECTORY=$(PWD)/build \
		OAS_DIRECTORY=$(PWD)/openapi \
		VERSION=$(VERSION) \
		go run . --config ./test/config/local.json

image:
	$(call cecho,Building Kerberos Docker image...,$(BOLD_YELLOW))
	docker build --build-arg VERSION=$(VERSION) -t ghcr.io/trebent/kerberos:$(VERSION) .

docker-run: image docker-stop docker-rm
	$(call cecho,Running Kerberos Docker container...,$(BOLD_YELLOW))
	docker run -d \
		-p $(KERBEROS_PORT):$(KERBEROS_PORT) \
		-p $(KERBEROS_ADMIN_PORT):$(KERBEROS_ADMIN_PORT) \
		-p $(KERBEROS_METRICS_PORT):$(KERBEROS_METRICS_PORT) \
		-e PORT=$(KERBEROS_PORT) \
		-e ADMIN_PORT=$(KERBEROS_ADMIN_PORT) \
		-e LOG_TO_CONSOLE=true \
		-e LOG_VERBOSITY=$(LOG_VERBOSITY) \
		-e OAS_DIRECTORY=/krb-oas \
		-v $(PWD)/test/config:/config:ro \
		-v $(PWD)/test/oas:/oas:ro \
		-v $(PWD)/openapi:/krb-oas:ro \
		--name kerberos \
		ghcr.io/trebent/kerberos:$(VERSION) \
		--config /config/docker.json

docker-stop:
	@docker stop kerberos || true

docker-rm:
	@docker rm kerberos || true

docker-logs:
	@docker logs kerberos

compose:
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
		docker compose -f test/compose/compose.yaml up -d --force-recreate

compose-wait:
	$(call cecho,Waiting for Kerberos to be ready...,$(BOLD_YELLOW))
	@until [ "$$(curl -s -o /dev/null -w '%{http_code}' localhost:$(KERBEROS_ADMIN_PORT)/api/admin/flow)" = "401" ]; do \
		echo "Waiting for Kerberos admin API..."; \
		sleep 1; \
	done; \
	echo "Kerberos is ready!"

compose-logs:
	@VERSION=$(VERSION) \
		KERBEROS_PORT=$(KERBEROS_PORT) \
		KERBEROS_ADMIN_PORT=$(KERBEROS_ADMIN_PORT) \
		KERBEROS_METRICS_PORT=$(KERBEROS_METRICS_PORT) \
		PROM_PORT=$(PROM_PORT) \
		GRAFANA_PORT=$(GRAFANA_PORT) \
		ECHO_PORT=$(ECHO_PORT) \
		ECHO_METRICS_PORT=$(ECHO_METRICS_PORT) \
		docker compose -f test/compose/compose.yaml logs kerberos echo protected-echo

compose-logs-f:
	@VERSION=$(VERSION) \
		KERBEROS_PORT=$(KERBEROS_PORT) \
		KERBEROS_ADMIN_PORT=$(KERBEROS_ADMIN_PORT) \
		KERBEROS_METRICS_PORT=$(KERBEROS_METRICS_PORT) \
		PROM_PORT=$(PROM_PORT) \
		GRAFANA_PORT=$(GRAFANA_PORT) \
		ECHO_PORT=$(ECHO_PORT) \
		ECHO_METRICS_PORT=$(ECHO_METRICS_PORT) \
		docker compose -f test/compose/compose.yaml logs -f kerberos echo protected-echo

compose-ps:
	@VERSION=$(VERSION) \
		KERBEROS_PORT=$(KERBEROS_PORT) \
		KERBEROS_ADMIN_PORT=$(KERBEROS_ADMIN_PORT) \
		KERBEROS_METRICS_PORT=$(KERBEROS_METRICS_PORT) \
		PROM_PORT=$(PROM_PORT) \
		GRAFANA_PORT=$(GRAFANA_PORT) \
		ECHO_PORT=$(ECHO_PORT) \
		ECHO_METRICS_PORT=$(ECHO_METRICS_PORT) \
		docker compose -f test/compose/compose.yaml ps

compose-down:
	$(call cecho,Tearing down Kerberos test environment...,$(BOLD_YELLOW))
	@VERSION=$(VERSION) \
		KERBEROS_PORT=$(KERBEROS_PORT) \
		KERBEROS_ADMIN_PORT=$(KERBEROS_ADMIN_PORT) \
		KERBEROS_METRICS_PORT=$(KERBEROS_METRICS_PORT) \
		PROM_PORT=$(PROM_PORT) \
		GRAFANA_PORT=$(GRAFANA_PORT) \
		ECHO_PORT=$(ECHO_PORT) \
		ECHO_METRICS_PORT=$(ECHO_METRICS_PORT) \
		docker compose -f test/compose/compose.yaml down

echo-build:
	$(call cecho,Building Echo binary...,$(BOLD_YELLOW))
	@CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o build/echo ./cmd/echo

echo-run:
	$(call cecho,Running echo...,$(BOLD_YELLOW))
	VERSION=$(VERSION) \
		go run ./cmd/echo

echo-image:
	$(call cecho,Building Echo Docker image...,$(BOLD_YELLOW))
	@docker build --build-arg VERSION=$(VERSION) \
		-f cmd/echo/Dockerfile \
		-t ghcr.io/trebent/kerberos/echo:$(VERSION) \
		.

echo-docker-run: echo-image echo-docker-stop echo-docker-rm
	$(call cecho,Running Echo Docker container...,$(BOLD_YELLOW))
	@docker run -d \
		-p $(ECHO_PORT):$(ECHO_PORT) \
		-p $(ECHO_METRICS_PORT):$(ECHO_METRICS_PORT) \
		-e VERSION=$(VERSION) \
		--name echo \
		ghcr.io/trebent/kerberos/echo:$(VERSION)

echo-docker-stop:
	@docker stop echo || true

echo-docker-rm:
	@docker rm echo || true

echo-docker-logs:
	@docker logs echo

#
# TEST
#

test-echo:
	$(call cecho,Sending a test request to echo...,$(BOLD_YELLOW))
	curl -X GET -i localhost:$(KERBEROS_PORT)/gw/backend/echo/hi

test-protected-echo:
	$(call cecho,Sending a test request to protected-echo...,$(BOLD_YELLOW))
	curl -X GET -i localhost:$(KERBEROS_PORT)/gw/backend/protected-echo/hi

get-flow:
	$(call cecho,Fetching flow from Kerberos admin API...,$(BOLD_YELLOW))
	@SESSION=$$(curl -s -o /dev/null -D - -X POST localhost:$(KERBEROS_ADMIN_PORT)/api/admin/superuser/login \
		-H "Content-Type: application/json" \
		-d '{"clientId":"$(SUPERUSER_CLIENT_ID)","clientSecret":"$(SUPERUSER_CLIENT_SECRET)"}' \
		| grep -i '^x-krb-session:' | tr -d '\r' | awk '{print $$2}'); \
	curl -s -H "x-krb-session: $$SESSION" localhost:$(KERBEROS_ADMIN_PORT)/api/admin/flow | jq .

get-oas-backend:
	$(call cecho,Fetching OAS backend from Kerberos admin API...,$(BOLD_YELLOW))
	@SESSION=$$(curl -s -o /dev/null -D - -X POST localhost:$(KERBEROS_ADMIN_PORT)/api/admin/superuser/login \
		-H "Content-Type: application/json" \
		-d '{"clientId":"$(SUPERUSER_CLIENT_ID)","clientSecret":"$(SUPERUSER_CLIENT_SECRET)"}' \
		| grep -i '^x-krb-session:' | tr -d '\r' | awk '{print $$2}'); \
	curl -s -H "x-krb-session: $$SESSION" localhost:$(KERBEROS_ADMIN_PORT)/api/admin/oas/echo

test-echo-methods:
	$(call cecho,Generating test HTTP requests for the echo backend...,$(BOLD_YELLOW))
	curl -X GET -i localhost:$(KERBEROS_PORT)/gw/backend/echo/hi
	curl -X PUT -i localhost:$(KERBEROS_PORT)/gw/backend/echo/hi
	curl -X POST -i localhost:$(KERBEROS_PORT)/gw/backend/echo/hi
	curl -X PATCH -i localhost:$(KERBEROS_PORT)/gw/backend/echo/hi
	curl -X DELETE -i localhost:$(KERBEROS_PORT)/gw/backend/echo/hi
	curl -X OPTIONS -i localhost:$(KERBEROS_PORT)/gw/backend/echo/hi
