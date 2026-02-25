BINARY_NAME=btfp
PREFIX=$(HOME)/go/bin

.PHONY: all build clean install uninstall test help

all: build

help:
	@echo "BTFP Makefile"
	@echo "Usage:"
	@echo "  make build       - Compile the binary"
	@echo "  make install     - Install binary to $(PREFIX) and setup Waybar"
	@echo "  make uninstall   - Remove binary from $(PREFIX)"
	@echo "  make test        - Run all tests"
	@echo "  make clean       - Remove local build artifacts"

build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) main.go

test:
	@echo "Running tests..."
	go test ./...

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
