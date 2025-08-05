.PHONY: test test-unit test-integration test-all coverage lint

# Run unit tests only (default)
test: test-unit

# Run unit tests without integration tests
test-unit:
	go test -v ./... -short

# Run integration tests (requires API keys)
test-integration:
	go test -v ./... -tags=integration

# Run all tests
test-all: test-unit test-integration

# Run tests with coverage
coverage:
	go test -v ./... -short -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# Run linter
lint:
	golangci-lint run

# Clean test artifacts
clean:
	rm -f coverage.out coverage.html

# Install dependencies
deps:
	go mod download
	go mod tidy

# Build the package
build:
	go build ./...

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...