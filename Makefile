PREFIX=$(HOME)/go/bin
GOLANGCI_LINT_VERSION=v1.64.5

SERVICES = btfp btfp-core btfp-library btfp-fetcher btfp-viz btfp-playlist

.PHONY: all build clean install uninstall test lint lint-install help

all: build

help:
	@echo "BTFP Microservices Makefile"
	@echo "Usage:"
	@echo "  make build       - Compile all service binaries"
	@echo "  make install     - Install all binaries to $(PREFIX) and setup Waybar"
	@echo "  make test        - Run linting and all tests"
	@echo "  make clean       - Remove local build artifacts"

build:
	@for service in $(SERVICES); do \
		echo "Building $$service..."; \
		go build -o $$service cmd/$$service/main.go; \
	done

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
	rm -f $(SERVICES)

install: build
	@echo "Stopping running microservices..."
	-pkill -x btfp-core btfp-library btfp-fetcher btfp-viz btfp-playlist || true
	@echo "Installing to $(PREFIX)..."
	mkdir -p $(PREFIX)
	@for service in $(SERVICES); do \
		install -m 755 $$service $(PREFIX)/$$service; \
	done
	@echo "Initializing configuration..."
	mkdir -p $(HOME)/.config/btfp/themes
	@echo "Setting up Waybar integration..."
	bash scripts/setup_waybar.sh $(PREFIX)/btfp
	@echo "Installation complete. Ensure $(PREFIX) is in your PATH."

uninstall:
	@echo "Uninstalling from $(PREFIX)..."
	for service in $(SERVICES); do \
		rm -f $(PREFIX)/$$service; \
	done
	@echo "Uninstall complete."
