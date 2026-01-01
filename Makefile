GRAFANA_PORT ?= 3000
PROM_PORT ?= 9090
KERBEROS_PORT ?= 30000
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

VERSION ?= $(shell git describe --tags --always)

define cecho
	@echo -e "${2}${1}${RESET}"
endef

default: lint vulncheck go-build

lint:
	$(call cecho,Running linter for Kerberos...,$(BOLD_YELLOW))
	@golangci-lint run --fix
	$(call cecho,Linter complete.,$(BOLD_GREEN))

codegen:
	$(call cecho,Running codegen for Kerberos...,$(BOLD_YELLOW))
	@go generate ./...
	$(call cecho,Codegen complete.,$(BOLD_GREEN))

unittest:
	$(call cecho,Running unit tests for Kerberos...,$(BOLD_YELLOW))
	@mkdir -p build
	@go test -v ./... -coverprofile=build/coverage.out
	@go tool cover -html=build/coverage.out -o build/coverage.html
	@go tool cover -func=build/coverage.out
	$(call cecho,Unit tests complete.,$(BOLD_GREEN))

total-coverage:
	@go tool cover -func=build/coverage.out | awk 'END {print $$3}'

functiontest:
	$(call cecho,Running function tests for Kerberos...,$(BOLD_YELLOW))
	@cd test/function && go test -v ./... -count=1
	$(call cecho,Function tests complete.,$(BOLD_GREEN))

vulncheck:
	$(call cecho,Running vulnerability check for Kerberos...,$(BOLD_YELLOW))
	@go tool govulncheck ./...
	$(call cecho,Vulnerability check complete.,$(BOLD_GREEN))

staticcheck: lint unittest vulncheck
	$(call cecho,Static analysis complete.,$(BOLD_GREEN))

go-build:
	$(call cecho,Building Kerberos binary...,$(BOLD_YELLOW))
	@mkdir -p build
	CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o build/kerberos ./cmd/kerberos
	$(call cecho,Build complete.,$(BOLD_GREEN))

run:
	$(call cecho,Running Kerberos...,$(BOLD_YELLOW))
	@OTEL_METRICS_EXPORTER=prometheus \
		OTEL_EXPORTER_PROMETHEUS_PORT=$(KERBEROS_METRICS_PORT) \
		LOG_TO_CONSOLE=true \
		LOG_VERBOSITY=100 \
		go run ./cmd/kerberos

docker-build:
	$(call cecho,Building Kerberos Docker image...,$(BOLD_YELLOW))
	docker build --build-arg VERSION=$(VERSION) -t ghcr.io/trebent/kerberos:$(VERSION) .
	$(call cecho,Docker image build complete.,$(BOLD_GREEN))

docker-run:
	$(call cecho,Running Kerberos Docker container...,$(BOLD_YELLOW))
	docker run -d \
		-p $(KERBEROS_PORT):$(KERBEROS_PORT) \
		-p $(KERBEROS_METRICS_PORT):$(KERBEROS_METRICS_PORT) \
		-e PORT=$(KERBEROS_PORT) \
		-e OTEL_METRICS_EXPORTER=prometheus \
		-e OTEL_EXPORTER_PROMETHEUS_PORT=$(KERBEROS_METRICS_PORT) \
		-e LOG_TO_CONSOLE=true \
		-e LOG_VERBOSITY=100 \
		--name kerberos \
		ghcr.io/trebent/kerberos:$(VERSION)
	$(call cecho,Kerberos Docker container is running.,$(BOLD_GREEN))

docker-stop:
	$(call cecho,Stopping Kerberos Docker container...,$(BOLD_YELLOW))
	@docker stop kerberos
	$(call cecho,Kerberos Docker container has been stopped.,$(BOLD_GREEN))

docker-rm:
	$(call cecho,Removing Kerberos Docker container...,$(BOLD_YELLOW))
	@docker rm kerberos
	$(call cecho,Kerberos Docker container has been removed.,$(BOLD_GREEN))

docker-logs:
	@docker logs -f kerberos

docker-compose-up:
	$(call cecho,Composing Kerberos test environment...,$(BOLD_YELLOW))
	VERSION=$(VERSION) \
		KERBEROS_PORT=$(KERBEROS_PORT) \
		KERBEROS_METRICS_PORT=$(KERBEROS_METRICS_PORT) \
		PROM_PORT=$(PROM_PORT) \
		GRAFANA_PORT=$(GRAFANA_PORT) \
		ECHO_PORT=$(ECHO_PORT) \
		ECHO_METRICS_PORT=$(ECHO_METRICS_PORT) \
		docker compose -f test/compose/compose.yaml up -d --force-recreate
	$(call cecho,Kerberos test environment is running.,$(BOLD_GREEN))

docker-compose-logs:
	VERSION=$(VERSION) \
		KERBEROS_PORT=$(KERBEROS_PORT) \
		KERBEROS_METRICS_PORT=$(KERBEROS_METRICS_PORT) \
		PROM_PORT=$(PROM_PORT) \
		GRAFANA_PORT=$(GRAFANA_PORT) \
		ECHO_PORT=$(ECHO_PORT) \
		ECHO_METRICS_PORT=$(ECHO_METRICS_PORT) \
		docker compose -f test/compose/compose.yaml logs kerberos echo

docker-compose-logs-f:
	VERSION=$(VERSION) \
		KERBEROS_PORT=$(KERBEROS_PORT) \
		KERBEROS_METRICS_PORT=$(KERBEROS_METRICS_PORT) \
		PROM_PORT=$(PROM_PORT) \
		GRAFANA_PORT=$(GRAFANA_PORT) \
		ECHO_PORT=$(ECHO_PORT) \
		ECHO_METRICS_PORT=$(ECHO_METRICS_PORT) \
		docker compose -f test/compose/compose.yaml logs -f kerberos echo

docker-compose-ps:
	VERSION=$(VERSION) \
		KERBEROS_PORT=$(KERBEROS_PORT) \
		KERBEROS_METRICS_PORT=$(KERBEROS_METRICS_PORT) \
		PROM_PORT=$(PROM_PORT) \
		GRAFANA_PORT=$(GRAFANA_PORT) \
		ECHO_PORT=$(ECHO_PORT) \
		ECHO_METRICS_PORT=$(ECHO_METRICS_PORT) \
		docker compose -f test/compose/compose.yaml ps

docker-compose-down:
	$(call cecho,Tearing down Kerberos test environment...,$(BOLD_YELLOW))
	VERSION=$(VERSION) \
		KERBEROS_PORT=$(KERBEROS_PORT) \
		KERBEROS_METRICS_PORT=$(KERBEROS_METRICS_PORT) \
		PROM_PORT=$(PROM_PORT) \
		GRAFANA_PORT=$(GRAFANA_PORT) \
		ECHO_PORT=$(ECHO_PORT) \
		ECHO_METRICS_PORT=$(ECHO_METRICS_PORT) \
		docker compose -f test/compose/compose.yaml down
	$(call cecho,Kerberos test environment has been torn down.,$(BOLD_GREEN))

echo-build:
	$(call cecho,Building Echo binary...,$(BOLD_YELLOW))
	@CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o build/echo ./cmd/echo
	$(call cecho,Echo build complete.,$(BOLD_GREEN))

echo-docker-build:
	$(call cecho,Building Echo Docker image...,$(BOLD_YELLOW))
	@docker build --build-arg VERSION=$(VERSION) -f cmd/echo/Dockerfile -t ghcr.io/trebent/kerberos/echo:$(VERSION) .
	$(call cecho,Echo Docker image build complete.,$(BOLD_GREEN))

echo-docker-run:
	$(call cecho,Running Echo Docker container...,$(BOLD_YELLOW))
	docker run -d \ 
		-p $(ECHO_PORT):$(ECHO_PORT) \
		-p $(ECHO_METRICS_PORT):$(ECHO_METRICS_PORT) \
		-e OTEL_METRICS_EXPORTER=prometheus \
		-e OTEL_EXPORTER_PROMETHEUS_PORT=$(ECHO_METRICS_PORT) \
		--name echo \
		ghcr.io/trebent/kerberos/echo:$(VERSION)
	$(call cecho,Echo Docker container is running.,$(BOLD_GREEN))

#
# TEST
#

generate-echo-test-requests:
	$(call cecho,Generating test HTTP requests for the echo backend...,$(BOLD_YELLOW))
	curl -X GET -I localhost:$(KERBEROS_PORT)/gw/backend/echo/test
	curl -X PUT -I localhost:$(KERBEROS_PORT)/gw/backend/echo/test
	curl -X POST -I localhost:$(KERBEROS_PORT)/gw/backend/echo/test
	curl -X PATCH -I localhost:$(KERBEROS_PORT)/gw/backend/echo/test
	curl -X DELETE -I localhost:$(KERBEROS_PORT)/gw/backend/echo/test
	curl -X OPTIONS -I localhost:$(KERBEROS_PORT)/gw/backend/echo/test

generate-test-requests:
	$(call cecho,Generating test HTTP requests to Kerberos...,$(BOLD_YELLOW))
	curl -X GET -I localhost:$(KERBEROS_PORT)/test
	curl -X GET -I localhost:$(KERBEROS_PORT)/test?status_code=400
	curl -X PUT -I localhost:$(KERBEROS_PORT)/test
	curl -X PUT -I localhost:$(KERBEROS_PORT)/test?status_code=500
	curl -X POST -I localhost:$(KERBEROS_PORT)/test
	curl -X POST -I localhost:$(KERBEROS_PORT)/test?status_code=204
	curl -X PATCH -I localhost:$(KERBEROS_PORT)/test
	curl -X PATCH -I localhost:$(KERBEROS_PORT)/test?status_code=404
	curl -X DELETE -I localhost:$(KERBEROS_PORT)/test
	curl -X OPTIONS -I localhost:$(KERBEROS_PORT)/test
