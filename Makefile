build:
	@echo "Building Kerberos..."
	go build -o kerberos ./cmd/main.go
	@echo "Build complete."

validate:
	@echo "Validating Kerberos..."
	@go tool govulncheck ./...
	@go tool golangci-lint run
	@go test -v ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Validation complete."