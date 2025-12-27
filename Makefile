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

default: validate build

lint:
	$(call cecho,Running linter for Kerberos...,$(BOLD_YELLOW))
	@golangci-lint run --fix
	$(call cecho,Linter complete.,$(BOLD_GREEN))

unittest:
	$(call cecho,Running unit tests for Kerberos...,$(BOLD_YELLOW))
	@go test -v ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	$(call cecho,Unit tests complete.,$(BOLD_GREEN))

functiontest:
	$(call cecho,Running functional tests for Kerberos...,$(BOLD_YELLOW))
	@go test -v ./test/functional/...
	$(call cecho,Functional tests complete.,$(BOLD_GREEN))

vulncheck:
	$(call cecho,Running vulnerability check for Kerberos...,$(BOLD_YELLOW))
	@go tool govulncheck ./...
	$(call cecho,Vulnerability check complete.,$(BOLD_GREEN))

staticcheck: lint unittest vulncheck
	$(call cecho,Static analysis complete.,$(BOLD_GREEN))

build:
	$(call cecho,Building Kerberos binary...,$(BOLD_YELLOW))
	CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o build/kerberos ./cmd/kerberos
	$(call cecho,Build complete.,$(BOLD_GREEN))

docker-build:
	$(call cecho,Building Kerberos Docker image...,$(BOLD_YELLOW))
	docker build --build-arg VERSION=$(VERSION) -t github.com/trebent/kerberos:$(VERSION) .
	$(call cecho,Docker image build complete.,$(BOLD_GREEN))

docker-run:
	$(call cecho,Running Kerberos Docker container...,$(BOLD_YELLOW))
	docker run -d \ 
		-p $(KERBEROS_PORT):$(KERBEROS_PORT) \
		-p $(KERBEROS_METRICS_PORT):$(KERBEROS_METRICS_PORT) \
		-e PORT=$(KERBEROS_PORT) \
		-e OTEL_METRICS_EXPORTER=prometheus \
		-e OTEL_EXPORTER_PROMETHEUS_PORT=$(KERBEROS_METRICS_PORT) \
		--name kerberos \
		github.com/trebent/kerberos:$(VERSION)
	$(call cecho,Kerberos Docker container is running.,$(BOLD_GREEN))

docker-compose-up:
	$(call cecho,Composing Kerberos test environment...,$(BOLD_YELLOW))
	docker compose -f test/compose/compose.yaml up -d --force-recreate
	$(call cecho,Kerberos test environment is running.,$(BOLD_GREEN))

docker-compose-down:
	$(call cecho,Tearing down Kerberos test environment...,$(BOLD_YELLOW))
	docker compose -f test/compose/compose.yaml down
	$(call cecho,Kerberos test environment has been torn down.,$(BOLD_GREEN))

echo-build:
	$(call cecho,Building Echo binary...,$(BOLD_YELLOW))
	CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o build/echo ./cmd/echo
	$(call cecho,Echo build complete.,$(BOLD_GREEN))

echo-docker-build:
	$(call cecho,Building Echo Docker image...,$(BOLD_YELLOW))
	docker build --build-arg VERSION=$(VERSION) -f cmd/echo/Dockerfile -t github.com/trebent/kerberos/echo:$(VERSION) .
	$(call cecho,Echo Docker image build complete.,$(BOLD_GREEN))

echo-docker-run:
	$(call cecho,Running Echo Docker container...,$(BOLD_YELLOW))
	docker run -d \ 
		-p $(ECHO_PORT):$(ECHO_PORT) \
		-p $(ECHO_METRICS_PORT):$(ECHO_METRICS_PORT) \
		-e OTEL_METRICS_EXPORTER=prometheus \
		-e OTEL_EXPORTER_PROMETHEUS_PORT=$(ECHO_METRICS_PORT) \
		--name echo \
		github.com/trebent/kerberos/echo:$(VERSION)
	$(call cecho,Echo Docker container is running.,$(BOLD_GREEN))

#
# TEST
#

generate-test-requests:
	$(call cecho,Generating test HTTP requests to Kerberos...,$(BOLD_YELLOW))
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
