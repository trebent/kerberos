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
	CGO_ENABLED=0 go build -trimpath -ldflags="-X 'github.com/trebent/kerberos/internal/version.Version=${VERSION}' -s -w" -o kerberos ./cmd/kerberos
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
		--name kerberos \
		github.com/trebent/kerberos:$(VERSION)
	$(call cecho,Kerberos Docker container is running.,$(BOLD_GREEN))

#
# TEST
#

setup-observability-env:
	$(call cecho,Setting up local observability services...,$(BOLD_YELLOW))
	@GRAFANA_PORT=$(GRAFANA_PORT) \
	PROM_PORT=$(PROM_PORT) \
	KERBEROS_PORT=$(KERBEROS_PORT) \
	KERBEROS_METRICS_PORT=$(KERBEROS_METRICS_PORT) \
	ECHO_PORT=$(ECHO_PORT) \
	ECHO_METRICS_PORT=$(ECHO_METRICS_PORT) \
	docker compose -f test/observability/compose.yaml up -d --force-recreate
	
	$(call cecho,Local observability deployment is up and running.,$(BOLD_GREEN))
	$(call cecho,Access Grafana at: http://localhost:$(GRAFANA_PORT) (default user: admin, password: admin),$(BOLD_GREEN))
	$(call cecho,Access Prometheus at: http://localhost:$(PROM_PORT),$(BOLD_GREEN))
	$(call cecho,Kerberos service is running at: http://localhost:$(KERBEROS_PORT),$(BOLD_GREEN))
	$(call cecho,Kerberos metrics are exposed at: http://localhost:$(KERBEROS_METRICS_PORT)/metrics,$(BOLD_GREEN))
	$(call cecho,Echo service is running at: http://localhost:$(ECHO_PORT),$(BOLD_GREEN))
	$(call cecho,Echo metrics are exposed at: http://localhost:$(ECHO_METRICS_PORT)/metrics,$(BOLD_GREEN))

teardown-observability-env:
	$(call cecho,Tearing down local observability services...,$(BOLD_YELLOW))
	docker compose -f test/observability/compose.yaml down
	$(call cecho,Local observability deployment has been torn down.,$(BOLD_GREEN))

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
