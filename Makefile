# Binary name
BINARY_NAME=ssht

# Build info
VERSION?=dev
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILT_BY=$(shell whoami)

# Build flags
LDFLAGS=-ldflags="-s -w -X github.com/akunbeben/ssht/cmd.Version=$(VERSION) -X github.com/akunbeben/ssht/cmd.Commit=$(COMMIT) -X github.com/akunbeben/ssht/cmd.Date=$(DATE) -X github.com/akunbeben/ssht/cmd.BuiltBy=$(BUILT_BY)"

.PHONY: all build run clean fmt tidy install release-snapshot release-check help

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	@go build $(LDFLAGS) -o $(BINARY_NAME) .

run: build
	@./$(BINARY_NAME)

clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME)
	@go clean

fmt:
	@echo "Formatting code..."
	@go fmt ./...

tidy:
	@echo "Cleaning up dependencies..."
	@go mod tidy

install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo cp $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "Done. You can now run '$(BINARY_NAME)' from anywhere."

release-snapshot:
	@echo "Performing snapshot release (dry-run)..."
	@goreleaser release --snapshot --clean

release-check:
	@echo "Checking goreleaser configuration..."
	@goreleaser check

help:
	@echo "Available commands:"
	@echo "  make build   - Build the binary"
	@echo "  make run     - Build and run the application"
	@echo "  make clean   - Remove the binary and temporary files"
	@echo "  make fmt     - Format source code"
	@echo "  make tidy    - Tidy up go.mod and go.sum"
	@echo "  make install - Install the binary to /usr/local/bin (requires sudo)"
	@echo "  make release-snapshot - Perform a dry-run release locally (requires goreleaser)"
	@echo "  make release-check - Check goreleaser configuration"
