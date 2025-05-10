# The main entry point is cmd/cli/main.go
APP_DIR = ./cmd/cli
APP_NAME = muserstory
BIN_DIR = ./bin
APP_PATH = $(BIN_DIR)/$(APP_NAME) # Full path to the binary

# Default target: build the application
.PHONY: all
all: build

# Build target: compiles the Go application
.PHONY: build
build:
	@echo "Building $(APP_NAME) into $(BIN_DIR)..."
	# Create the bin directory if it doesn't exist
	mkdir -p $(BIN_DIR)
	# Use go build to compile the main package
	# -o specifies the output file name and path
	# $(APP_DIR) specifies the directory containing the main package
	go build -o $(APP_PATH) $(APP_DIR)
	@echo "Build complete."

# Run target: builds and runs the application
.PHONY: run
run: build
	@echo "Running $(APP_NAME) from $(BIN_DIR)..."
	# Execute the built binary
	./$(APP_PATH)

# Clean target: removes the built binary and any other generated files
.PHONY: clean
clean:
	@echo "Cleaning up build artifacts..."
	# Use go clean to remove object files and cached files
	go clean
	# Remove the built binary directory
	rm -rf $(BIN_DIR)
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

