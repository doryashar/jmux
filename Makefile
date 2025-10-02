.PHONY: all build build-go build-jcat clean test help install

# Default target
all: build

# Build both jcat and jmux-go
build: build-jcat build-go

# Build Go jmux version
build-go:
	@echo "Building jmux Go version..."
	@./scripts/build-jmux-go.sh

# Build jcat binary
build-jcat:
	@echo "Building jcat..."
	@./scripts/build-jcat.sh

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f bin/jcat-binary bin/jmux-go
	@rm -f src/jcat/jcat

# Run tests
test:
	@echo "Running tests..."
	@cd tests && ./test_jmux.sh

# Install to system (requires sudo)
install: build
	@echo "Installing jmux (bash), jmux-go, and jcat to /usr/local/bin..."
	@sudo cp bin/jmux /usr/local/bin/jmux-bash
	@sudo cp bin/jmux-go /usr/local/bin/jmux
	@sudo cp bin/jcat-binary /usr/local/bin/jcat
	@sudo chmod +x /usr/local/bin/jmux-bash /usr/local/bin/jmux /usr/local/bin/jcat
	@echo "✓ Installed successfully"
	@echo "  jmux        -> Go version (default)"
	@echo "  jmux-bash   -> Bash version"
	@echo "  jcat        -> jcat binary"

# Show help
help:
	@echo "Available targets:"
	@echo "  build      - Build both jcat and jmux-go"
	@echo "  build-go   - Build jmux Go version only"
	@echo "  build-jcat - Build jcat binary only"
	@echo "  clean      - Clean build artifacts"
	@echo "  test       - Run tests"
	@echo "  install    - Install to system (requires sudo)"
	@echo "  help       - Show this help message"