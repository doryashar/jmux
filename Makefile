.PHONY: all build clean test help install

# Default target
all: build

# Build jcat binary
build:
	@echo "Building jcat..."
	@./scripts/build-jcat.sh

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f bin/jcat-binary
	@rm -f src/jcat/jcat

# Run tests
test:
	@echo "Running tests..."
	@cd tests && ./test_jmux.sh

# Install to system (requires sudo)
install: build
	@echo "Installing jmux and jcat to /usr/local/bin..."
	@sudo cp bin/jmux /usr/local/bin/
	@sudo cp bin/jcat-binary /usr/local/bin/jcat
	@sudo chmod +x /usr/local/bin/jmux /usr/local/bin/jcat
	@echo "✓ Installed successfully"

# Show help
help:
	@echo "Available targets:"
	@echo "  build    - Build jcat binary"
	@echo "  clean    - Clean build artifacts"
	@echo "  test     - Run tests"
	@echo "  install  - Install to system (requires sudo)"
	@echo "  help     - Show this help message"