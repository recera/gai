.PHONY: all build test clean lint fmt vet security benchmark coverage help

# Variables
BINARY_NAME=gai
GO=go
GOFLAGS=-v
COVERPROFILE=coverage.txt

# Default target
all: clean lint test build

## help: Show this help message
help:
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## build: Build the project
build:
	$(GO) build $(GOFLAGS) ./...

## test: Run all tests
test:
	$(GO) test -v -race ./...

## test-short: Run short tests
test-short:
	$(GO) test -v -short ./...

## coverage: Run tests with coverage
coverage:
	$(GO) test -v -race -coverprofile=$(COVERPROFILE) -covermode=atomic ./...
	$(GO) tool cover -html=$(COVERPROFILE) -o coverage.html
	@echo "Coverage report generated: coverage.html"

## benchmark: Run benchmarks
benchmark:
	$(GO) test -bench=. -benchmem -run=^$$ ./...

## lint: Run linters
lint: vet
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install with:"; \
		echo "  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin"; \
	fi

## vet: Run go vet
vet:
	$(GO) vet ./...

## fmt: Format code
fmt:
	$(GO) fmt ./...
	gofmt -s -w .

## clean: Clean build artifacts
clean:
	$(GO) clean
	rm -f $(COVERPROFILE) coverage.html
	rm -rf dist/

## security: Run security checks
security:
	@echo "Running govulncheck..."
	@if command -v govulncheck > /dev/null; then \
		govulncheck ./...; \
	else \
		echo "govulncheck not installed. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
	fi
	@echo "Running gosec..."
	@if command -v gosec > /dev/null; then \
		gosec -quiet ./...; \
	else \
		echo "gosec not installed. Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
	fi

## deps: Download dependencies
deps:
	$(GO) mod download
	$(GO) mod verify

## tidy: Tidy go.mod
tidy:
	$(GO) mod tidy

## update: Update dependencies
update:
	$(GO) get -u ./...
	$(GO) mod tidy

## install: Install the binary
install:
	$(GO) install ./...

## ci: Run CI checks locally
ci: clean deps lint security test coverage

## watch: Watch for changes and run tests
watch:
	@if command -v watchexec > /dev/null; then \
		watchexec -c -r -e go -- make test-short; \
	else \
		echo "watchexec not installed. Install from: https://github.com/watchexec/watchexec"; \
	fi

## doc: Generate and serve documentation
doc:
	@echo "Starting godoc server on http://localhost:6060"
	godoc -http=:6060

## check: Quick check before commit
check: fmt vet test-short

## release: Create a new release (requires VERSION parameter)
release:
ifndef VERSION
	$(error VERSION is not set. Usage: make release VERSION=v1.0.0)
endif
	@echo "Creating release $(VERSION)"
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)