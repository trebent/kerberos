GRAFANA_PORT ?= 3000
PROM_PORT ?= 9090
KERBEROS_PORT ?= 30000
KERBEROS_METRICS_PORT ?= 9464
ECHO_PORT ?= 15000
ECHO_METRICS_PORT ?= 9463

VERSION ?= $(shell git describe --tags --always)

default: validate build

validate:
	@echo "\033[0;32mValidating Kerberos...\033[0m"
	@go tool govulncheck ./...
	@golangci-lint run --fix
	@go test -v ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "\033[0;32mValidation complete.\033[0m"

build:
	@echo "\033[0;32mBuilding Kerberos...\033[0m"
	CGO_ENABLED=0 go build -trimpath -ldflags="-X 'github.com/trebent/kerberos/internal/version.Ver=${VERSION}' -s -w" -o kerberos ./cmd/kerberos
	@echo "\033[0;32mBuild complete.\033[0m"

docker-build:
	@echo "\033[0;32mBuilding Kerberos docker image...\033[0m"
	docker build --build-arg VERSION=$(VERSION) -t github.com/trebent/kerberos:$(VERSION) .
	@echo "\033[0;32mBuild complete.\033[0m"

#
# TEST
#

setup-observability-env:
	@echo "\033[0;32mComposing local observability services...\033[0m"
	@GRAFANA_PORT=$(GRAFANA_PORT) \
	PROM_PORT=$(PROM_PORT) \
	KERBEROS_PORT=$(KERBEROS_PORT) \
	KERBEROS_METRICS_PORT=$(KERBEROS_METRICS_PORT) \
	ECHO_PORT=$(ECHO_PORT) \
	ECHO_METRICS_PORT=$(ECHO_METRICS_PORT) \
	docker compose -f test/observability/compose.yaml up -d --force-recreate
	
	@echo "\033[0;32mComposed local observability deployment.\033[0m"
	@echo " - Grafana on port:          $(GRAFANA_PORT)"
	@echo " - Prometheus on port:       $(PROM_PORT)"
	@echo " - Kerberos on port:         $(KERBEROS_PORT)"
	@echo " - Kerberos metrics on port: $(KERBEROS_METRICS_PORT)"
	@echo " - Echo on port:             $(ECHO_PORT)"
	@echo " - Echo metrics on port:     $(ECHO_METRICS_PORT)"

teardown-observability-env:
	@echo "\033[0;32mStopping local observability services...\033[0m"
	@GRAFANA_PORT=$(GRAFANA_PORT) \
	PROM_PORT=$(PROM_PORT) \
	KERBEROS_PORT=$(KERBEROS_PORT) \
	KERBEROS_METRICS_PORT=$(KERBEROS_METRICS_PORT) \
	ECHO_PORT=$(ECHO_PORT) \
	ECHO_METRICS_PORT=$(ECHO_METRICS_PORT) \
	docker compose -f test/observability/compose.yaml down
	@echo "\033[0;32mStopped local observability deployment.\033[0m"

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
