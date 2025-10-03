.PHONY: all build build-go build-jcat clean test help install release version

# Default target
all: build

# Build both jcat and dmux
build: build-jcat build-go

# Build Go dmux version
build-go:
	@echo "Building dmux Go version..."
	@./scripts/build-jmux-go.sh

# Build jcat binary
build-jcat:
	@echo "Building jcat..."
	@./scripts/build-jcat.sh

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f bin/jcat-binary bin/dmux
	@rm -f src/jcat/jcat

# Run tests
test:
	@echo "Running tests..."
	@cd tests && ./test_jmux.sh

# Install to system (requires sudo)
install: build
	@echo "Installing jmux (bash), dmux, and jcat to /usr/local/bin..."
	@sudo cp bin/jmux /usr/local/bin/jmux-bash
	@sudo cp bin/dmux /usr/local/bin/dmux
	@sudo cp bin/jcat-binary /usr/local/bin/jcat
	@sudo chmod +x /usr/local/bin/jmux-bash /usr/local/bin/dmux /usr/local/bin/jcat
	@echo "✓ Installed successfully"
	@echo "  dmux        -> Go version (default)"
	@echo "  jmux-bash   -> Bash version"
	@echo "  jcat        -> jcat binary"

# Create a release
release:
	@echo "Creating release..."
	@./scripts/release.sh

# Show version
version:
	@./bin/dmux version --verbose 2>/dev/null || echo "Build dmux first with 'make build-go'"

# Show help
help:
	@echo "Available targets:"
	@echo "  build      - Build both jcat and dmux"
	@echo "  build-go   - Build dmux Go version only"
	@echo "  build-jcat - Build jcat binary only"
	@echo "  clean      - Clean build artifacts"
	@echo "  test       - Run tests"
	@echo "  install    - Install to system (requires sudo)"
	@echo "  release    - Create a new release (tags, builds, uploads)"
	@echo "  version    - Show current version"
	@echo "  help       - Show this help message"