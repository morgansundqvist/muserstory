# The main entry point is cmd/cli/main.go
APP_DIR = ./cmd/cli
APP_NAME = myapp # You can change this to your desired binary name

# Default target: build the application
.PHONY: all
all: build

# Build target: compiles the Go application
.PHONY: build
build:
	@echo "Building $(APP_NAME)..."
	# Use go build to compile the main package
	# -o specifies the output file name
	# $(APP_DIR) specifies the directory containing the main package
	go build -o $(APP_NAME) $(APP_DIR)
	@echo "Build complete."

# Run target: builds and runs the application
.PHONY: run
run: build
	@echo "Running $(APP_NAME)..."
	# Execute the built binary
	./$(APP_NAME)

# Clean target: removes the built binary and any other generated files
.PHONY: clean
clean:
	@echo "Cleaning up build artifacts..."
	# Use go clean to remove object files and cached files
	go clean
	# Remove the built binary
	rm -f $(APP_NAME)
	@echo "Clean complete."

# Test target: runs all tests in the project
# Note: This assumes you have test files (e.g., *_test.go) in your packages
.PHONY: test
test:
	@echo "Running tests..."
	# Use go test to run tests in all packages (./...)
	go test ./...
	@echo "Tests complete."

# Optional: Add a target for formatting Go code
.PHONY: fmt
fmt:
	@echo "Formatting Go code..."
	# Use go fmt to format all Go files
	go fmt ./...
	@echo "Formatting complete."

# Help target: displays available targets
.PHONY: help
help:
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-10s\033[0m %s\n", $$1, $$2}'

