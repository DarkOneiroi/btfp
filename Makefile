BINARY_NAME=btfp
PREFIX=$(HOME)/go/bin
GOLANGCI_LINT_VERSION=v1.64.5

.PHONY: all build clean install uninstall test lint lint-install help

all: build

help:
	@echo "BTFP Makefile"
	@echo "Usage:"
	@echo "  make build       - Compile the binary"
	@echo "  make install     - Install binary to $(PREFIX) and setup Waybar"
	@echo "  make uninstall   - Remove binary from $(PREFIX)"
	@echo "  make test        - Run linting and all tests"
	@echo "  make lint        - Run golangci-lint"
	@echo "  make clean       - Remove local build artifacts"

build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) main.go

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found. Install it or run 'make lint-install' for local installation."; \
		exit 1; \
	fi

lint-install:
	@echo "Installing golangci-lint locally..."
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin $(GOLANGCI_LINT_VERSION)

test: lint
	@echo "Running tests..."
	go test -v ./...

clean:
	@echo "Cleaning up..."
	rm -f $(BINARY_NAME)

install: build
	@echo "Installing to $(PREFIX)..."
	mkdir -p $(PREFIX)
	cp $(BINARY_NAME) $(PREFIX)/
	@echo "Initializing configuration..."
	mkdir -p $(HOME)/.config/btfp/themes
	@echo "Setting up Waybar integration..."
	bash scripts/setup_waybar.sh $(PREFIX)/$(BINARY_NAME)
	@echo "Installation complete. Ensure $(PREFIX) is in your PATH."

uninstall:
	@echo "Uninstalling from $(PREFIX)..."
	rm -f $(PREFIX)/$(BINARY_NAME)
	@echo "Uninstall complete."
